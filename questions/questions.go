package questions

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v2"
)

type Questions struct {
	Questions []Question `yaml:"questions"`
}

type Question struct {
	Category         string   `yaml:"category"`
	Prompt           string   `yaml:"prompt"`
	Choices          []string `yaml:"choices"`
	CorrectAnswerIdx int      `yaml:"correct_answer_idx"`
}

var (
	m   = make(map[string][]Question)
	mut = sync.Mutex{}
)

func loadQuestionsFromYaml(filepath string) ([]Question, error) {
	q := &Questions{}
	if _, badFilePathErr := os.Stat(filepath); os.IsNotExist(badFilePathErr) {
		return q.Questions, badFilePathErr
	}
	f, openErr := os.Open(filepath)
	if openErr != nil {
		return q.Questions, openErr
	}

	bytes, readErr := ioutil.ReadAll(f)
	if readErr != nil {
		return q.Questions, readErr
	}

	unmarshalErr := yaml.Unmarshal(bytes, q)
	if unmarshalErr != nil {
		return q.Questions, unmarshalErr
	}
	return q.Questions, nil
}

func loadQuestionsIntoMemory(questions []Question) {
	for _, question := range questions {
		m[question.Category] = append(m[question.Category], question)
	}
}

func localYamlPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Println("Error looking up current directory")
	}
	return filepath.Join(cwd, "questions.yml")
}

func init() {
	q, err := loadQuestionsFromYaml(localYamlPath())
	if err != nil {
		fmt.Printf("Error loading questions from YAML: %s\n", err)
	}
	loadQuestionsIntoMemory(q)
}

func ListCategories() []string {
	categories := []string{}
	for category := range m {
		categories = append(categories, category)
	}
	return categories
}

func GetQuestionsByCategory(category string) []Question {
	return m[category]
}
