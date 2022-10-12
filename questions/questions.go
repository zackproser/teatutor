package questions

import "sync"

type Question struct {
	Prompt           string
	Choices          []string
	CorrectAnswerIdx int
}

var (
	m   = make(map[string][]Question)
	mut = sync.Mutex{}
)

func init() {
	loadInitialQuestions()
}

func loadInitialQuestions() {
	defer mut.Unlock()
	mut.Lock()

	m["IAM"] = []Question{
		{
			Prompt: "You need to provide AWS credentials to an EC2 instance so that an application running on the instance can contact the S3 and DynamoDB services. How should you provide AWS credentials to the instance?",
			Choices: []string{
				"Create an IAM role. Assign the role to an instance profile. Attach the instance profile to the EC2 instance",
				"Create an IAM user. Generate security credentials for the IAM user, then write them to ~/.aws/credentials on the EC2 instance",
				"SSH into the EC2 instance. Export the ${AWS_ACCESS_KEY_ID} and ${AWS_SECRET_ACCESS_KEY} environment variables so that the application running on the instance can contact the other AWS services",
			},
			CorrectAnswerIdx: 0,
		},
		{
			Prompt: "What should you do with the root account user?",
			Choices: []string{
				"Use it to resolve permissions issues that are preventing services from functioning properly in your AWS account",
				"Store its login credentials in a secure password manager. Then, never use the root user. Create an IAM user and attach Administrator permissions to it. Use the IAM user for daily work",
				"Only log into when you need to administer sensitive account settings such as billing, address or payment information",
				"Use it for debugging services that might log sensitive information like usernames, passwords or social security numbers",
			},
			CorrectAnswerIdx: 1,
		},
	}

	m["Lambda"] = []Question{
		{
			Prompt:           "What is the maximum amount of time a lamdba function can run for?",
			Choices:          []string{"10 minutes", "15 minutes", "25 minutes"},
			CorrectAnswerIdx: 1,
		},

		{
			Prompt: "Which of the following are valid ways to trigger a lambda function other than a web request?",
			Choices: []string{
				"CloudWatch metric filters and EventBridge cron",
				"EventBridge schedule syntax",
				"DynamoDB streams and S3 buckets",
				"All of the above",
			},
			CorrectAnswerIdx: 3,
		},
	}

	m["S3"] = []Question{
		{
			Prompt:           "Can you use S3 buckets to host a static web site?",
			Choices:          []string{"Yes", "No"},
			CorrectAnswerIdx: 0,
		},

		{
			Prompt:           "Should you leak sensitive secrets by uploading them to a public S3 bucket?",
			Choices:          []string{"Yes", "No"},
			CorrectAnswerIdx: 1,
		},
	}
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
