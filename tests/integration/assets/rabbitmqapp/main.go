/*
Copyright 2021 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/streadway/amqp"
)

func main() {
	uriStr := os.Getenv("DATABASE_URI")
	log.Printf("Connecting to %q\n", uriStr)
	conn, err := amqp.Dial(uriStr)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal(err)
	}
	defer ch.Close()

	queue, err := ch.QueueDeclare(
		"foo", // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatal(err)
	}

	expectedValue := "Hello World!"

	go func() {
		if err := ch.Publish(
			"",         // exchange
			queue.Name, // routing key
			false,      // mandatory
			false,      // immediate
			amqp.Publishing{
				ContentType: "text/plain",
				Body:        []byte(expectedValue),
			},
		); err != nil {
			log.Fatal(err)
		}
	}()

	msgs, err := ch.Consume(
		queue.Name, // queue
		"",         // consumer
		true,       // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		log.Fatal(err)
	}

	msg := <-msgs
	value := string(msg.Body)

	if value != expectedValue {
		log.Fatal(fmt.Errorf("Value %q is not the expected %q", value, expectedValue))
	}

	port, exists := os.LookupEnv("PORT")
	if !exists {
		port = "8080"
	}

	handler := http.NewServeMux()

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: handler,
	}

	stop := make(chan struct{}, 1)

	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "ok")
		log.Println("Successfully received request!")
		stop <- struct{}{}
	})

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	select {
	case <-stop:
		os.Exit(0)
	case <-time.After(30 * time.Second):
		log.Fatal("Error: timed out")
	}
}
