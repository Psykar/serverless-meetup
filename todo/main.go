package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// Response is of type APIGatewayProxyResponse since we're leveraging the
// AWS Lambda Proxy Request functionality (default behavior)
//
// https://serverless.com/framework/docs/providers/aws/events/apigateway/#lambda-proxy-integration
type Response events.APIGatewayProxyResponse

const tableName = "todos"
const todoKey = "name"

// Handler is our lambda handler invoked by the `lambda.Start` function call
func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (Response, error) {
	var buf bytes.Buffer

	var message string
	message = req.Path
	statusCode := 200

	// Could use a third party routing library at this point, but being hacky for now
	items := strings.Split(req.Path, "/")
	var item string
	if len(items) > 1 {
		item = strings.Join(items[2:], "/")
	}

	// If we actually have an action to take
	if len(items) >= 1 {
		switch items[1] {
		case "list":
			items, err := List()
			if err != nil {
				statusCode = 500
				message = fmt.Sprint(err)
			} else {
				message = strings.Join(items, "\n")
			}
		case "add":
			// Should probably be doing this on PUT or POST only
			err := Add(item)
			if err != nil {
				statusCode = 500
				message = fmt.Sprint(err)
			} else {
				message = "Added"
			}

		case "complete":
			// Should only be doing this on POST, but demo
			err := Complete(item)
			if err != nil {
				statusCode = 500
				message = fmt.Sprint(err)
			} else {
				message = "Completed"
			}
		}
	}

	body, err := json.Marshal(map[string]interface{}{
		"message": message,
	})
	if err != nil {
		return Response{StatusCode: 404}, err
	}
	json.HTMLEscape(&buf, body)

	resp := Response{
		StatusCode:      statusCode,
		IsBase64Encoded: false,
		Body:            buf.String(),
		Headers: map[string]string{
			"Content-Type":           "application/json",
			"X-MyCompany-Func-Reply": "hello-handler",
		},
	}

	return resp, nil
}

// List the items in the todo
func List() ([]string, error) {
	mySession, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	svc := dynamodb.New(mySession)

	input := &dynamodb.ScanInput{}
	input.SetTableName(tableName)
	scanOutput, err := svc.Scan(input)
	if err != nil {
		return nil, err
	}

	results := []string{}
	for _, row := range scanOutput.Items {
		// If the dynamodb item might have something not a string this would have to change
		results = append(results, *row[todoKey].S)
	}
	return results, nil
}

// Add an item to the todo list
func Add(v string) error {
	mySession, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	svc := dynamodb.New(mySession)

	item := &dynamodb.PutItemInput{}
	attrs := map[string]*dynamodb.AttributeValue{
		todoKey: &dynamodb.AttributeValue{
			S: &v,
		},
	}
	item.SetItem(attrs)
	item.SetTableName(tableName)

	_, err = svc.PutItem(item)
	return err
}

// Complete an item in the todo list (just delete it at this stage...)
func Complete(v string) error {
	mySession, err := session.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	svc := dynamodb.New(mySession)

	item := &dynamodb.DeleteItemInput{}
	item.SetTableName(tableName)
	key := map[string]*dynamodb.AttributeValue{
		todoKey: &dynamodb.AttributeValue{
			S: &v,
		},
	}
	item.SetKey(key)

	_, err = svc.DeleteItem(item)
	return err

}

func main() {
	lambda.Start(Handler)
}
