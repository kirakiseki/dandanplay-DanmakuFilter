package utils

import (
	"encoding/xml"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

type RuleFile struct {
	Content []byte
	Type    string
}

type Rule struct {
	Type string
	Rule string
}

type RuleXML struct {
	XMLName xml.Name `xml:"filters"`
	Text    string   `xml:",chardata"`
	Item    []struct {
		Text    string `xml:",chardata"`
		Enabled string `xml:"enabled,attr"`
	} `xml:"item"`
}

func ReadRules() []RuleFile {
	rulesDir := os.Getenv("RULES")
	if rulesDir == "" {
		Inst.Logger.Warn().Msg("RULES environment variable not set or empty, using default value: /rules")
		rulesDir = "/rules"
	}

	var result []RuleFile
	var files []string

	err := filepath.Walk(rulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, rulesDir+"/"+info.Name())
		}
		return nil
	})

	if err != nil {
		Inst.Logger.Fatal().Err(err).Msg("Failed to read rules")
	}

	for _, file := range files {
		if file != rulesDir {
			data, err := ioutil.ReadFile(file)
			if err != nil {
				Inst.Logger.Fatal().Err(err).Msg("Failed to read rules")
			}

			Inst.Logger.Info().Msg("Read rule file: " + file)

			if path.Ext(file) == ".txt" {
				result = append(result, RuleFile{Content: data, Type: "txt"})
			} else if path.Ext(file) == ".xml" {
				result = append(result, RuleFile{Content: data, Type: "xml"})
			} else {
				Inst.Logger.Warn().Msg("Invalid rule file extension: " + file)
			}
		}
	}

	return result
}

func ParseRules(rules []RuleFile) []Rule {
	var result []Rule

	for _, rule := range rules {
		if rule.Type == "txt" {
			result = append(result, ParseTxtRule(rule.Content)...)
		} else if rule.Type == "xml" {
			result = append(result, ParseXmlRule(rule.Content)...)
		} else {
			Inst.Logger.Warn().Msg("Invalid rule type: " + rule.Type)
		}
	}

	return result
}

func ParseTxtRule(content []byte) []Rule {
	var rules []Rule

	emojiRx := regexp.MustCompile(`\\u\w{4}`)

	for _, line := range strings.Split(string(content), "\n") {
		line = emojiRx.ReplaceAllString(line, `[e]`)
		rules = append(rules, Rule{Type: "keyword", Rule: line})
	}

	return rules
}

func ParseXmlRule(content []byte) []Rule {
	var ruleXML RuleXML
	var rules []Rule

	err := xml.Unmarshal(content, &ruleXML)
	if err != nil {
		Inst.Logger.Fatal().Err(err).Msg("Failed to parse XML rule")
	}

	emojiRx := regexp.MustCompile(`\\u\w{4}`)

	for _, item := range ruleXML.Item {
		if item.Enabled == "true" && strings.HasPrefix(item.Text, "r=") {
			item.Text = emojiRx.ReplaceAllString(item.Text, `[e]`)

			rules = append(rules, Rule{Type: "regex", Rule: strings.TrimPrefix(item.Text, "r=")})
		}
	}

	return rules
}
