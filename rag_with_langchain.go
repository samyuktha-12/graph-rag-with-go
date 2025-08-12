package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

func startRAGChatbot() {
	// Initialize Neo4j connection
	driver, err := neo4j.NewDriver("bolt://localhost:7687", neo4j.BasicAuth("neo4j", "", ""))
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}
	defer driver.Close()

	// Initialize LLM for query generation
	llm, err := ollama.New(ollama.WithModel("llama3.2"))
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Get graph schema for context
	schema := getGraphSchema(driver)

	// Interactive chat loop
	fmt.Println("🤖 Marvel Comics RAG Chatbot (LLM-Powered)")
	fmt.Println("Ask me about Marvel characters, their relationships, and comic appearances!")
	fmt.Println("Type 'quit' to exit.")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("You: ")
		scanner.Scan()
		userInput := strings.TrimSpace(scanner.Text())

		if strings.ToLower(userInput) == "quit" {
			fmt.Println("Goodbye! 🦸‍♂️")
			break
		}

		// Generate Cypher query using LLM
		cypherQuery, err := generateCypherQuery(llm, userInput, schema)
		if err != nil {
			fmt.Printf("❌ Error generating query: %v\n", err)
			continue
		}

		fmt.Printf("🔍 Generated Cypher query:\n%s\n", cypherQuery)

		// Execute query and get results
		results := executeQuery(driver, cypherQuery)
		fmt.Printf("📊 Results:\n%s\n\n", results)
	}
}

func getGraphSchema(driver neo4j.Driver) string {
	session := driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	// Get node labels and their properties
	result, err := session.Run(`
		CALL db.labels() YIELD label
		RETURN collect(label) as labels
	`, nil)
	if err != nil {
		return "Graph schema unavailable"
	}

	var labels []string
	if result.Next() {
		record := result.Record()
		if len(record.Values) > 0 {
			labelsInterface := record.Values[0].([]interface{})
			for _, label := range labelsInterface {
				labels = append(labels, label.(string))
			}
		}
	}

	// Get relationship types
	result, err = session.Run(`
		CALL db.relationshipTypes() YIELD relationshipType
		RETURN collect(relationshipType) as relationships
	`, nil)
	if err != nil {
		return "Graph schema unavailable"
	}

	var relationships []string
	if result.Next() {
		record := result.Record()
		if len(record.Values) > 0 {
			relationshipsInterface := record.Values[0].([]interface{})
			for _, rel := range relationshipsInterface {
				relationships = append(relationships, rel.(string))
			}
		}
	}

	return fmt.Sprintf("Node labels: %v, Relationship types: %v", labels, relationships)
}

func generateCypherQuery(llm llms.Model, userQuery, schema string) (string, error) {
	prompt := fmt.Sprintf(`You are a Cypher query generator for a Neo4j Marvel Comics knowledge graph.

Graph Schema:
%s

CRITICAL DATA STRUCTURE:
- Character nodes: (c:Character {id: string, name: string, group: string, size: int})
- Hero nodes: (h:Hero {id: string, name: string})
- Comic nodes: (c:Comic {id: string, title: string})
- Relationships: (c1:Character)-[:PARTNERS_WITH]->(c2:Character), (h1:Hero)-[:KNOWS]->(h2:Hero), (h:Hero)-[:APPEARS_IN]->(c:Comic)

MANDATORY RULES - FOLLOW EXACTLY:
1. ALWAYS use c.id, h.id, c.id for ALL property access
2. NEVER use c.name, h.name, c.title
3. Use single quotes for strings: 'Iron Man'
4. Use EXACT matches: {id: 'Character Name'} or WHERE c.id IN ['Name1', 'Name2']
5. NEVER use toLower() or CONTAINS - only exact matches
6. Always include LIMIT 10
7. Return a single string column named 'result'
8. Keep queries SIMPLE - avoid complex logic
9. For counting: use WITH count(*) as count, then toString(count) in RETURN
10. NEVER use colons in RETURN strings - use + for concatenation

User Question: "%s"

Choose the appropriate pattern and return ONLY the Cypher query:

For character partnerships (like "who are spider-man's partners?"):
MATCH (c:Character {id: 'Spider-Man'}) OPTIONAL MATCH (c)-[:PARTNERS_WITH]->(partner:Character) WITH c, collect(DISTINCT partner.id) as partners RETURN 'Character: ' + c.id + ', Partners: ' + partners as result LIMIT 10

For Avengers teammates (like "which avengers have fought together?"):
MATCH (c1:Character)-[:PARTNERS_WITH]->(c2:Character) WHERE c1.id IN ['Iron Man', 'Captain America', 'Thor', 'Hulk', 'Black Widow', 'Hawkeye'] AND c2.id IN ['Iron Man', 'Captain America', 'Thor', 'Hulk', 'Black Widow', 'Hawkeye'] RETURN 'Avengers teammates: ' + c1.id + ' and ' + c2.id as result LIMIT 10

For exact character match (like "who are iron man's partners?"):
MATCH (c:Character {id: 'Iron Man'}) OPTIONAL MATCH (c)-[:PARTNERS_WITH]->(partner:Character) WITH c, collect(DISTINCT partner.id) as partners RETURN 'Character: ' + c.id + ', Partners: ' + partners as result LIMIT 10

For character search (like "find spider-man"):
MATCH (c:Character {id: 'Spider-Man'}) RETURN 'Character: ' + c.id + ', Group: ' + c.group as result LIMIT 10

For counting relationships (like "how many does X know?"):
MATCH (h:Hero {id: 'Human Robot'})-[:KNOWS]->(other:Hero) WITH count(other) as count RETURN 'Human Robot knows ' + toString(count) + ' heroes' as result LIMIT 10

For counting partnerships (like "how many avengers partnerships?"):
MATCH (c1:Character)-[:PARTNERS_WITH]->(c2:Character) WHERE c1.id IN ['Iron Man', 'Captain America', 'Thor', 'Hulk', 'Black Widow', 'Hawkeye'] AND c2.id IN ['Iron Man', 'Captain America', 'Thor', 'Hulk', 'Black Widow', 'Hawkeye'] WITH count(*) as count RETURN 'There are ' + toString(count) + ' Avengers partnerships' as result LIMIT 10

For cross-team partnerships (like "how many avengers are partners with Spider-Man?"):
MATCH (c1:Character)-[:PARTNERS_WITH]->(c2:Character) WHERE c1.id IN ['Iron Man', 'Captain America', 'Thor', 'Hulk', 'Black Widow', 'Hawkeye'] AND c2.id = 'Spider-Man' WITH count(c1) as count RETURN 'There are ' + toString(count) + ' Avengers partnered with Spider-Man' as result LIMIT 10

Only return the Cypher query, nothing else.`, schema, userQuery)

	ctx := context.Background()
	response, err := llm.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	})
	if err != nil {
		return "", fmt.Errorf("LLM generation failed: %v", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("empty response from LLM")
	}

	cypherQuery := strings.TrimSpace(response.Choices[0].Content)

	// Basic validation - ensure it's a Cypher query
	if !strings.Contains(strings.ToUpper(cypherQuery), "MATCH") {
		return "", fmt.Errorf("generated query doesn't contain MATCH clause")
	}

	return cypherQuery, nil
}

func executeQuery(driver neo4j.Driver, cypherQuery string) string {
	session := driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	result, err := session.Run(cypherQuery, nil)
	if err != nil {
		return fmt.Sprintf("❌ Query execution error: %v", err)
	}

	var results []string
	for result.Next() {
		record := result.Record()
		values := record.Values
		if len(values) > 0 {
			results = append(results, fmt.Sprintf("%v", values[0]))
		}
	}

	if len(results) == 0 {
		return "❌ No results found."
	}

	return strings.Join(results, "\n")
}
