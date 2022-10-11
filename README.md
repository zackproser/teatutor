# Bubbletea AWS Certification Quiz App
Better name forthcoming. This is a [bubbletea](github.com/charmbracelet/bubbletea) program designed to be run locally or over ssh 

It helps you study for AWS certification exams by providing a high quality set of questions and an easy to navigate, slide-deck-like interface. 

When you complete your 

# Usage 

## Run locally 

`go run main.go` 

## Run in server mode (to host over ssh)
```bash 
# export the special env var to enable server mode 
export QUIZ_SERVER=true 
go run main.go
```

## Connect to server as a client 

`ssh -p 23234 <ip-address-of-server>`

There is no auth required - all ssh public keys are accepted. 
