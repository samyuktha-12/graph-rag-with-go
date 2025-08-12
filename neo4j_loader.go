package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func loadDataToNeo4j() {
	// 1. Connect to Neo4j
	driver, err := neo4j.NewDriver("bolt://localhost:7687", neo4j.BasicAuth("neo4j", "", ""))
	if err != nil {
		log.Fatalf("Failed to create driver: %v", err)
	}
	defer driver.Close()

	// 2. Find all CSVs in dataset folder
	var csvFiles []string
	err = filepath.Walk("dataset", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".csv") {
			csvFiles = append(csvFiles, path)
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to read datasets folder: %v", err)
	}

	session := driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite})
	defer session.Close()

	// 3. Clear existing data
	clearDatabase(session)

	// 4. Create unified schema
	createSchema(session)

	// 5. Load nodes first, then relationships
	loadNodesFirst(session, csvFiles)
	loadRelationships(session, csvFiles)

	fmt.Println("✅ All datasets loaded into one unified Neo4j knowledge graph.")
}

func clearDatabase(session neo4j.Session) {
	_, err := session.Run("MATCH (n) DETACH DELETE n", nil)
	if err != nil {
		log.Fatalf("Failed to clear database: %v", err)
	}
	fmt.Println("🗑️ Database cleared.")
}

func loadNodesFirst(session neo4j.Session, csvFiles []string) {
	for _, filePath := range csvFiles {
		fileName := filepath.Base(filePath)
		if strings.Contains(strings.ToLower(fileName), "nodes") {
			fmt.Printf("📂 Loading nodes: %s\n", filePath)
			loadCSVIntoNeo4j(session, filePath)
		}
	}
}

func loadRelationships(session neo4j.Session, csvFiles []string) {
	for _, filePath := range csvFiles {
		fileName := filepath.Base(filePath)
		if strings.Contains(strings.ToLower(fileName), "edges") || strings.Contains(strings.ToLower(fileName), "hero-network") {
			fmt.Printf("📂 Loading relationships: %s\n", filePath)
			loadCSVIntoNeo4j(session, filePath)
		}
	}
}

func createSchema(session neo4j.Session) {
	constraints := []string{
		"CREATE CONSTRAINT IF NOT EXISTS FOR (c:Character) REQUIRE c.id IS UNIQUE",
		"CREATE CONSTRAINT IF NOT EXISTS FOR (h:Hero) REQUIRE h.id IS UNIQUE",
		"CREATE CONSTRAINT IF NOT EXISTS FOR (c:Comic) REQUIRE c.id IS UNIQUE",
		"CREATE CONSTRAINT IF NOT EXISTS FOR (m:Movie) REQUIRE m.id IS UNIQUE",
		"CREATE CONSTRAINT IF NOT EXISTS FOR (t:Team) REQUIRE t.id IS UNIQUE",
	}
	for _, constraint := range constraints {
		_, err := session.Run(constraint, nil)
		if err != nil {
			log.Fatalf("Failed to create constraint: %v", err)
		}
	}
	fmt.Println("✅ Graph schema ready.")
}

func loadCSVIntoNeo4j(session neo4j.Session, filePath string) {
	fileName := filepath.Base(filePath)
	dirName := filepath.Base(filepath.Dir(filePath))

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open %s: %v", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read CSV %s: %v", filePath, err)
	}
	if len(records) < 2 {
		fmt.Printf("⚠️ Skipping empty file: %s\n", fileName)
		return
	}
	data := records[1:]

	// === Marvel Characters Partnerships ===
	if strings.Contains(strings.ToLower(dirName), "marvel_characters_partnerships") {
		if strings.Contains(strings.ToLower(fileName), "nodes") {
			for _, row := range data {
				if len(row) >= 3 {
					size, _ := strconv.Atoi(row[2])
					_, err := session.Run(`
						MERGE (c:Character {id: $id})
						SET c.name = $id, c.group = $group, c.size = $size
					`, map[string]interface{}{
						"id":    row[1],
						"group": row[0],
						"size":  size,
					})
					if err != nil {
						log.Printf("Failed MERGE character %s: %v", row[1], err)
					}
				}
			}
			fmt.Println("✅ Characters loaded from partnerships dataset.")
		} else if strings.Contains(strings.ToLower(fileName), "edges") {
			for _, row := range data {
				if len(row) >= 2 {
					_, err := session.Run(`
						MATCH (a:Character {id: $source})
						MATCH (b:Character {id: $target})
						MERGE (a)-[:PARTNERS_WITH]->(b)
					`, map[string]interface{}{
						"source": row[0],
						"target": row[1],
					})
					if err != nil {
						log.Printf("Failed MERGE relationship %s -> %s: %v", row[0], row[1], err)
					}
				}
			}
			fmt.Println("✅ Partnerships loaded.")
		}
		return
	}

	// === Marvel Universe Social Network ===
	if strings.Contains(strings.ToLower(dirName), "marvel_universe_social_network") {
		if strings.Contains(strings.ToLower(fileName), "nodes") {
			for _, row := range data {
				if len(row) >= 2 {
					if row[1] == "hero" {
						_, err := session.Run(`
							MERGE (h:Hero {id: $id})
							SET h.name = $id
						`, map[string]interface{}{"id": row[0]})
						if err != nil {
							log.Printf("Failed MERGE hero %s: %v", row[0], err)
						}
					} else if row[1] == "comic" {
						_, err := session.Run(`
							MERGE (c:Comic {id: $id})
							SET c.title = $id
						`, map[string]interface{}{"id": row[0]})
						if err != nil {
							log.Printf("Failed MERGE comic %s: %v", row[0], err)
						}
					}
				}
			}
			fmt.Println("✅ Heroes & Comics loaded from social network dataset.")
		} else if strings.Contains(strings.ToLower(fileName), "hero-network") {
			for _, row := range data {
				if len(row) >= 2 {
					_, err := session.Run(`
						MATCH (h1:Hero {id: $hero1})
						MATCH (h2:Hero {id: $hero2})
						MERGE (h1)-[:KNOWS]->(h2)
					`, map[string]interface{}{
						"hero1": row[0],
						"hero2": row[1],
					})
					if err != nil {
						log.Printf("Failed MERGE hero link %s -> %s: %v", row[0], row[1], err)
					}
				}
			}
			fmt.Println("✅ Hero-to-hero links loaded.")
		} else if strings.Contains(strings.ToLower(fileName), "edges") {
			for _, row := range data {
				if len(row) >= 2 {
					_, err := session.Run(`
						MATCH (h:Hero {id: $hero})
						MATCH (c:Comic {id: $comic})
						MERGE (h)-[:APPEARS_IN]->(c)
					`, map[string]interface{}{
						"hero":  row[0],
						"comic": row[1],
					})
					if err != nil {
						log.Printf("Failed MERGE appearance %s -> %s: %v", row[0], row[1], err)
					}
				}
			}
			fmt.Println("✅ Hero-to-comic appearances loaded.")
		}
		return
	}

	fmt.Printf("⚠️ Skipping unrecognized dataset: %s\n", fileName)
}
