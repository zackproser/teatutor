# Tea Tutor 

![Tea Tutor SSH Quiz application](./docs/logo.png)

This is a [bubbletea](github.com/charmbracelet/bubbletea) program designed to be run locally or over ssh. 


# Demos

[![Tea Tutor Demo](http://img.youtube.com/vi/Dk2neG9vp84/0.jpg)](http://www.youtube.com/watch?v=Dk2neG9vp84 "Tea Tutor Demo")

Early version recorded via Asciinema:  
* [https://asciinema.org/a/6G1YbJG1W0nWa6bSkyPpotxk4](https://asciinema.org/a/6G1YbJG1W0nWa6bSkyPpotxk4)


# App at a glance

Tea tutor helps you study for AWS certification exams by providing a high quality set of questions and an easy to navigate, slide-deck-like interface. 


![Bubbletea Quiz Over SSH](./docs/intro.png)
When you first load the program, you are presented with a list of current study tracks to choose from. 

![Choose a study category](./docs/categories.png)

Choose a study topic to begin.

![Bubbletea quiz slide deck](./docs/quiz1_000.png)

You'll be asked a series of multiple choice and true or false questions that test your AWS product knowledge. You can page back and forth between questions and change your answers as needed. 

![Bubbletea quiz advancing](./docs/quiz2_000.png)

When you complete your quiz, you'll be shown the results page that reviews the questions you answered, your responses and whether they were correct. 

![Results report](./docs/quiz-results2.png)


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
