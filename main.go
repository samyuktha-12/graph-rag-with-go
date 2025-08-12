package main

import (
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
	// Connect to Neo4j
	uri := "neo4j://localhost:7687" // Bolt protocol
	username := "neo4j"
	password := ""

	driver, err := neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		log.Fatalf("Failed to create driver: %v", err)
	}
	defer driver.Close()

	// Open a session
	session := driver.NewSession(neo4j.SessionConfig{})
	defer session.Close()

	// Run a write transaction
	_, err = session.WriteTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		_, err := tx.Run(
			"CREATE (p:Person {name: $name}) RETURN p",
			map[string]interface{}{"name": "Samyuktha"},
		)
		return nil, err
	})
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}

	// Run a read transaction
	result, err := session.ReadTransaction(func(tx neo4j.Transaction) (interface{}, error) {
		res, err := tx.Run(
			"MATCH (p:Person) RETURN p.name AS name",
			nil,
		)
		if err != nil {
			return nil, err
		}

		var names []string
		for res.Next() {
			names = append(names, res.Record().Values[0].(string))
		}
		return names, res.Err()
	})
	if err != nil {
		log.Fatalf("Failed to read node: %v", err)
	}

	fmt.Println("People in database:", result)
}
