# 🤖 Marvel Comics Graph RAG with Go

A powerful **Retrieval-Augmented Generation (RAG)** system that combines **Neo4j Graph Database** with **LLM-powered natural language queries** to explore Marvel Comics character relationships and partnerships.

## 🌟 Features

- **🔍 LLM-Powered Query Generation** - Natural language to Cypher query conversion
- **📊 Graph Database Integration** - Neo4j for storing Marvel character relationships
- **🎨 Beautiful Web UI** - Dark theme interface similar to Claude
- **🤖 Natural Language Responses** - Human-friendly explanations of graph results
- **📱 Responsive Design** - Works on desktop and mobile devices
- **⚡ Real-time Status Monitoring** - Database, LLM, and data loading status
- **🔄 Smart Data Detection** - Automatically detects if data is already loaded

## 🏗️ Architecture

```
User Query → LLM (Cypher Generation) → Neo4j Database → LLM (Natural Response) → Web UI
```

## 📋 Prerequisites

- **Go** (version 1.19 or higher)
- **Neo4j** (version 5.x or higher)
- **Ollama** (for local LLM inference)

## 🚀 Quick Start

### 1. Install Dependencies

#### Install Go
```bash
brew install go
go version
```

#### Install Neo4j
```bash
brew install neo4j
brew services start neo4j
```

#### Install Ollama
```bash
brew install ollama
ollama pull llama3.2
```

### 2. Set Up Neo4j

1. **Start Neo4j:**
   ```bash
   brew services start neo4j
   ```

2. **Access Neo4j Browser:**
   - Open: http://localhost:7474
   - Default credentials: `neo4j` / `neo4j`
   - Change password to: `YOUR PASSWORD`

3. **Restart Neo4j with new credentials:**
   ```bash
   brew services restart neo4j
   ```

### 3. Initialize Go Project

```bash
# Initialize Go module
go mod init graph-rag-with-go

# Install dependencies
go get github.com/neo4j/neo4j-go-driver/v5/neo4j
go get github.com/tmc/langchaingo/llms
go get github.com/tmc/langchaingo/llms/ollama
```

### 4. Test Neo4j Connection

Create a test file to verify the connection:

```go
package main

import (
    "fmt"
    "log"
    "github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func main() {
    // Connect to Neo4j
    uri := "neo4j://localhost:7687"
    username := "neo4j"
    password := "Samyuktha@12"

    driver, err := neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""))
    if err != nil {
        log.Fatalf("Failed to create driver: %v", err)
    }
    defer driver.Close()

    // Test connection
    session := driver.NewSession(neo4j.SessionConfig{})
    defer session.Close()

    result, err := session.Run("RETURN 'Hello Neo4j!' as message", nil)
    if err != nil {
        log.Fatalf("Failed to run query: %v", err)
    }

    if result.Next() {
        fmt.Println("✅ Neo4j connection successful:", result.Record().Values[0])
    }
}
```

Run the test:
```bash
go run test_connection.go
```

### 5. Download Datasets

Create a `dataset` folder and download the Marvel Comics datasets:

```bash
mkdir dataset
cd dataset
```

#### Dataset 1: Marvel Universe Social Network
- **Source:** [Kaggle - The Marvel Universe Social Network](https://www.kaggle.com/datasets/csanhueza/the-marvel-universe-social-network)
- **Files:** `nodes.csv`, `hero-network.csv`, `edges.csv`

#### Dataset 2: Marvel Comic Characters Partnerships
- **Source:** [Kaggle - The Marvel Comic Characters Partnerships](https://www.kaggle.com/datasets/trnguyen1510/the-marvel-comic-characters-partnerships)
- **Files:** `nodes.csv`, `edges.csv`

### 6. Run the Application

```bash
go run main.go neo4j_loader.go rag_with_langchain.go web_ui.go
```

Open your browser and navigate to: **http://localhost:8080**

## 🎯 Usage

### Web Interface

1. **Load Data** - Click the "📊 Load Data" button to populate the database
2. **Ask Questions** - Use natural language to query the Marvel knowledge graph
3. **View Results** - Get natural language responses with optional technical details

### Example Queries

- "Who are Spider-Man's partners?"
- "Which Avengers have fought together?"
- "How many Avengers partnerships are there?"
- "Find Captain America"
- "Who are Iron Man's partners?"
- "How many Avengers are partners with Spider-Man?"

## 🏗️ Project Structure

```
graph-rag-with-go/
├── main.go                 # Application entry point
├── neo4j_loader.go         # Data loading and Neo4j operations
├── rag_with_langchain.go   # LLM-powered query generation
├── web_ui.go              # Web interface and API endpoints
├── dataset/               # Marvel Comics datasets
│   ├── marvel_characters_partnerships/
│   │   ├── nodes.csv
│   │   └── edges.csv
│   └── marvel_universe_social_network/
│       ├── nodes.csv
│       ├── hero-network.csv
│       └── edges.csv
├── go.mod                 # Go module dependencies
└── README.md             # This file
```

## 🔧 Technical Details

### Database Schema

- **Character Nodes:** `(c:Character {id, name, group, size})`
- **Hero Nodes:** `(h:Hero {id, name})`
- **Comic Nodes:** `(c:Comic {id, title})`
- **Relationships:**
  - `(c1:Character)-[:PARTNERS_WITH]->(c2:Character)`
  - `(h1:Hero)-[:KNOWS]->(h2:Hero)`
  - `(h:Hero)-[:APPEARS_IN]->(c:Comic)`

### LLM Integration

- **Model:** Ollama with Llama 3.2
- **Query Generation:** Natural language → Cypher queries
- **Response Generation:** Graph results → Natural language explanations

### API Endpoints

- `GET /` - Web interface
- `POST /api/query` - Process natural language queries
- `GET /api/status` - Check system status
- `POST /api/load-data` - Load datasets into Neo4j

