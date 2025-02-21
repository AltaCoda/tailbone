package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jedib0t/go-pretty/v6/list"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/manifoldco/promptui"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

//goland:noinspection GoUnusedGlobalVariable
var (
	ListHeaderColor = text.Colors{text.FgHiGreen, text.Bold}
	LocalLocation   *time.Location
)

type OutData struct {
	List    list.Writer
	Headers table.Row
	Rows    []table.Row
	RawData []interface{}
}

var (
	defaultTableStyle = table.Style{
		Box: table.BoxStyle{
			PaddingLeft:  " ",
			PaddingRight: " ",
		},
		Options: table.OptionsNoBordersAndSeparators,
	}
	defaultListStyle = list.Style{
		CharItemSingle:   "",
		CharItemTop:      "",
		CharItemFirst:    "",
		CharItemMiddle:   "",
		CharItemVertical: "  ",
		CharItemBottom:   "",
		CharNewline:      "\n",
		LinePrefix:       "",
		Name:             "defaultStyle",
	}
)

func init() {
	LocalLocation = time.Now().Location()
}

func PrintJSON(data interface{}) (err error) {
	buf, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	fmt.Print(string(buf))

	return nil
}

func PrintYAML(data interface{}) (err error) {
	buf, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	fmt.Print(string(buf))

	return nil
}

func Print(data interface{}) (err error) {
	switch v := data.(type) {
	case OutData:
		return printOutData(v)
	default:
		switch viper.GetString("output") {
		case "json":
			return PrintJSON(data)
		case "yaml", "yml":
			return PrintYAML(data)
		default:
			return fmt.Errorf("unknown format: %s", viper.GetString("output"))
		}
	}
}

func PrintInfo(format string, args ...interface{}) {
	if IsText() {
		os.Stderr.WriteString(fmt.Sprintf(format, args...))
	}
}

func IsText() bool {
	return viper.GetString("output") == "text"
}

func GetTimeInUserZone(t time.Time) string {
	return t.In(LocalLocation).Format("2006-01-02 15:04:05")
}

func ExpectYes(label string) bool {
	if viper.GetBool("no-confirmation") {
		return true
	}

	prompt := promptui.Prompt{
		Label:     label,
		IsConfirm: true,
	}

	result, err := prompt.Run()
	if err != nil {
		return false
	}

	if strings.ToLower(result) != "y" {
		fmt.Println("Aborted")
		return false
	}

	return true
}

func printOutData(data OutData) error {
	switch viper.GetString("output") {
	case "json":
		return PrintJSON(data.RawData)
	case "yaml", "yml":
		return PrintYAML(data.RawData)
	default:
		if data.List == nil {
			dateHeaders := make([]int, 0)
			t := table.NewWriter()
			t.SetOutputMirror(os.Stdout)
			t.SetStyle(defaultTableStyle)
			t.AppendHeader(data.Headers)
			t.AppendRows(data.Rows)

			for idx, header := range data.Headers {
				if headerValue, ok := header.(string); ok {
					// if the header name ends in _AT then we assume it's a timestamp
					if len(headerValue) > 3 && headerValue[len(headerValue)-3:] == "_AT" {
						dateHeaders = append(dateHeaders, idx)
					}
				}
			}

			for _, row := range data.Rows {
				for _, idx := range dateHeaders {
					if row[idx] != nil {
						// is it time.Time
						if t, ok := row[idx].(time.Time); ok {
							row[idx] = t.In(LocalLocation).Format("2006-01-02 15:04:05")
						} else if t, ok := row[idx].(*time.Time); ok {
							if t == nil {
								continue
							}

							row[idx] = t.In(LocalLocation).Format("2006-01-02 15:04:05")
						} else {
							// leave it alone it might be a mislabeled column header ending in _AT
							continue
						}
					}
				}
			}

			t.Render()
		} else if data.Rows == nil {
			data.List.SetOutputMirror(os.Stdout)
			data.List.SetStyle(defaultListStyle)
			data.List.Render()
		} else {
			panic("both list and rows are populated")
		}

		return nil
	}
}
