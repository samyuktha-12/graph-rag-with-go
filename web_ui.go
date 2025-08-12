package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/ollama"
)

type QueryRequest struct {
	Query string `json:"query"`
}

type QueryResponse struct {
	Query     string `json:"query"`
	Cypher    string `json:"cypher"`
	Results   string `json:"results"`
	Response  string `json:"response"`
	Error     string `json:"error,omitempty"`
	Timestamp string `json:"timestamp"`
}

var (
	driver     neo4j.Driver
	llm        llms.Model
	schema     string
	dataLoaded bool
)

func startWebUI() {
	// Initialize Neo4j connection
	var err error
	driver, err = neo4j.NewDriver("bolt://localhost:7687", neo4j.BasicAuth("neo4j", "", ""))
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}
	defer driver.Close()

	// Initialize LLM
	llm, err = ollama.New(ollama.WithModel("llama3.2"))
	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
	}

	// Get graph schema
	schema = getGraphSchema(driver)

	// Check if data is already loaded
	dataLoaded = checkIfDataExists(driver)

	// Serve static files
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/api/query", handleQuery)
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/load-data", handleLoadData)

	fmt.Println("🌐 Starting Web UI...")
	fmt.Println("📱 Open your browser and go to: http://localhost:8080")
	fmt.Println("🎨 Dark theme UI with Claude-like interface")
	fmt.Println("🤖 LLM-powered Marvel Comics Knowledge Graph")
	fmt.Println()

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Marvel Comics RAG Chatbot</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: #0f0f23;
            color: #e6e6e6;
            line-height: 1.6;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            height: 100vh;
            display: flex;
            flex-direction: column;
        }

        .header {
            text-align: center;
            margin-bottom: 30px;
            padding: 20px;
            background: rgba(255, 255, 255, 0.05);
            border-radius: 12px;
            border: 1px solid rgba(255, 255, 255, 0.1);
        }

        .header h1 {
            font-size: 2.5rem;
            margin-bottom: 10px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }

        .header p {
            color: #a0a0a0;
            font-size: 1.1rem;
            margin-bottom: 20px;
        }

        .status-bar {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 15px 20px;
            background: rgba(255, 255, 255, 0.03);
            border-radius: 8px;
            border: 1px solid rgba(255, 255, 255, 0.1);
            margin-bottom: 20px;
        }

        .status-item {
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .status-indicator {
            width: 12px;
            height: 12px;
            border-radius: 50%;
            background: #ef4444;
        }

        .status-indicator.connected {
            background: #10b981;
        }

        .load-button {
            background: linear-gradient(135deg, #10b981 0%, #059669 100%);
            color: white;
            border: none;
            border-radius: 8px;
            padding: 10px 20px;
            font-size: 0.9rem;
            cursor: pointer;
            transition: all 0.2s ease;
            font-weight: 500;
        }

        .load-button:hover {
            transform: translateY(-1px);
            box-shadow: 0 4px 12px rgba(16, 185, 129, 0.3);
        }

        .load-button:disabled {
            opacity: 0.6;
            cursor: not-allowed;
            transform: none;
        }

        .chat-container {
            flex: 1;
            display: flex;
            flex-direction: column;
            background: rgba(255, 255, 255, 0.03);
            border-radius: 12px;
            border: 1px solid rgba(255, 255, 255, 0.1);
            overflow: hidden;
        }

        .chat-messages {
            flex: 1;
            padding: 20px;
            overflow-y: auto;
            max-height: 60vh;
        }

        .message {
            margin-bottom: 20px;
            padding: 15px 20px;
            border-radius: 12px;
            animation: fadeIn 0.3s ease-in;
        }

        .message.user {
            background: rgba(102, 126, 234, 0.2);
            border: 1px solid rgba(102, 126, 234, 0.3);
            margin-left: 20%;
        }

        .message.assistant {
            background: rgba(255, 255, 255, 0.05);
            border: 1px solid rgba(255, 255, 255, 0.1);
            margin-right: 20%;
        }

        .message.system {
            background: rgba(245, 158, 11, 0.1);
            border: 1px solid rgba(245, 158, 11, 0.3);
            text-align: center;
            margin: 0 10%;
        }

        .message-header {
            font-size: 0.9rem;
            color: #888;
            margin-bottom: 8px;
            font-weight: 500;
        }

        .message-content {
            font-size: 1rem;
            line-height: 1.6;
        }

        .cypher-query {
            background: rgba(0, 0, 0, 0.3);
            padding: 12px;
            border-radius: 8px;
            font-family: 'Monaco', 'Menlo', monospace;
            font-size: 0.9rem;
            color: #4ade80;
            margin: 10px 0;
            border-left: 3px solid #4ade80;
        }

        .natural-response {
            background: rgba(34, 197, 94, 0.1);
            padding: 15px;
            border-radius: 8px;
            font-size: 1rem;
            color: #4ade80;
            margin: 10px 0;
            border-left: 3px solid #4ade80;
            line-height: 1.6;
        }

        .results {
            background: rgba(59, 130, 246, 0.1);
            padding: 12px;
            border-radius: 8px;
            font-family: 'Monaco', 'Menlo', monospace;
            font-size: 0.9rem;
            color: #60a5fa;
            margin: 10px 0;
            border-left: 3px solid #60a5fa;
            white-space: pre-wrap;
            display: none;
        }

        .toggle-results {
            background: rgba(59, 130, 246, 0.2);
            color: #60a5fa;
            border: none;
            border-radius: 6px;
            padding: 8px 12px;
            font-size: 0.8rem;
            cursor: pointer;
            margin: 10px 0;
            transition: all 0.2s ease;
        }

        .toggle-results:hover {
            background: rgba(59, 130, 246, 0.3);
        }

        .error {
            background: rgba(239, 68, 68, 0.1);
            padding: 12px;
            border-radius: 8px;
            color: #f87171;
            margin: 10px 0;
            border-left: 3px solid #f87171;
        }

        .input-container {
            padding: 20px;
            background: rgba(255, 255, 255, 0.02);
            border-top: 1px solid rgba(255, 255, 255, 0.1);
        }

        .input-form {
            display: flex;
            gap: 10px;
            align-items: flex-end;
        }

        .input-field {
            flex: 1;
            background: rgba(255, 255, 255, 0.05);
            border: 1px solid rgba(255, 255, 255, 0.1);
            border-radius: 8px;
            padding: 12px 16px;
            color: #e6e6e6;
            font-size: 1rem;
            resize: none;
            min-height: 50px;
            max-height: 120px;
            font-family: inherit;
        }

        .input-field:focus {
            outline: none;
            border-color: #667eea;
            box-shadow: 0 0 0 2px rgba(102, 126, 234, 0.2);
        }

        .send-button {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 8px;
            padding: 12px 24px;
            font-size: 1rem;
            cursor: pointer;
            transition: all 0.2s ease;
            font-weight: 500;
        }

        .send-button:hover {
            transform: translateY(-1px);
            box-shadow: 0 4px 12px rgba(102, 126, 234, 0.3);
        }

        .send-button:disabled {
            opacity: 0.6;
            cursor: not-allowed;
            transform: none;
        }

        .loading {
            display: none;
            text-align: center;
            padding: 20px;
            color: #888;
        }

        .spinner {
            border: 2px solid rgba(255, 255, 255, 0.1);
            border-top: 2px solid #667eea;
            border-radius: 50%;
            width: 20px;
            height: 20px;
            animation: spin 1s linear infinite;
            margin: 0 auto 10px;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }

        @keyframes fadeIn {
            from { opacity: 0; transform: translateY(10px); }
            to { opacity: 1; transform: translateY(0); }
        }

        .examples {
            margin-top: 20px;
            padding: 20px;
            background: rgba(255, 255, 255, 0.02);
            border-radius: 12px;
            border: 1px solid rgba(255, 255, 255, 0.1);
        }

        .examples h3 {
            margin-bottom: 15px;
            color: #a0a0a0;
        }

        .example-queries {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 10px;
        }

        .example-query {
            background: rgba(255, 255, 255, 0.05);
            padding: 10px 15px;
            border-radius: 8px;
            cursor: pointer;
            transition: all 0.2s ease;
            border: 1px solid rgba(255, 255, 255, 0.1);
        }

        .example-query:hover {
            background: rgba(102, 126, 234, 0.1);
            border-color: rgba(102, 126, 234, 0.3);
        }

        .example-query.disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }

        @media (max-width: 768px) {
            .container {
                padding: 10px;
            }
            
            .header h1 {
                font-size: 2rem;
            }
            
            .message.user {
                margin-left: 10%;
            }
            
            .message.assistant {
                margin-right: 10%;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🤖 Marvel Comics RAG Chatbot</h1>
            <p>Ask me about Marvel characters, their relationships, and comic appearances!</p>
            
            <div class="status-bar">
                <div class="status-item">
                    <div class="status-indicator" id="neo4jStatus"></div>
                    <span>Neo4j Database</span>
                </div>
                <div class="status-item">
                    <div class="status-indicator" id="llmStatus"></div>
                    <span>LLM Model</span>
                </div>
                <div class="status-item">
                    <div class="status-indicator" id="dataStatus"></div>
                    <span>Data Loaded</span>
                </div>
                <button class="load-button" id="loadButton" onclick="loadData()">📊 Load Data</button>
            </div>
        </div>

        <div class="chat-container">
            <div class="chat-messages" id="chatMessages">
                <div class="message assistant">
                    <div class="message-header">🤖 Assistant</div>
                    <div class="message-content">
                        Hello! I'm your Marvel Comics Knowledge Graph assistant. I can help you explore relationships between characters, find partnerships, and discover connections in the Marvel universe. If data isn't loaded yet, use the "Load Data" button above.
                    </div>
                </div>
            </div>

            <div class="loading" id="loading">
                <div class="spinner"></div>
                <div>Generating response...</div>
            </div>

            <div class="input-container">
                <form class="input-form" id="queryForm">
                    <textarea 
                        class="input-field" 
                        id="queryInput" 
                        placeholder="Ask me about Marvel characters, their relationships, and comic appearances..."
                        rows="1"
                        disabled
                    ></textarea>
                    <button type="submit" class="send-button" id="sendButton" disabled>Send</button>
                </form>
            </div>
        </div>

        <div class="examples">
            <h3>💡 Example Queries</h3>
            <div class="example-queries">
                <div class="example-query disabled" onclick="setQuery('Who are Spider-Man\'s partners?')">Who are Spider-Man's partners?</div>
                <div class="example-query disabled" onclick="setQuery('Which Avengers have fought together?')">Which Avengers have fought together?</div>
                <div class="example-query disabled" onclick="setQuery('Who are Iron Man\'s partners?')">Who are Iron Man's partners?</div>
                <div class="example-query disabled" onclick="setQuery('How many Avengers partnerships are there?')">How many Avengers partnerships are there?</div>
                <div class="example-query disabled" onclick="setQuery('Find Captain America')">Find Captain America</div>
                <div class="example-query disabled" onclick="setQuery('How many Avengers are partners with Spider-Man?')">How many Avengers are partners with Spider-Man?</div>
            </div>
        </div>
    </div>

    <script>
        const chatMessages = document.getElementById('chatMessages');
        const queryForm = document.getElementById('queryForm');
        const queryInput = document.getElementById('queryInput');
        const sendButton = document.getElementById('sendButton');
        const loading = document.getElementById('loading');
        const loadButton = document.getElementById('loadButton');
        const neo4jStatus = document.getElementById('neo4jStatus');
        const llmStatus = document.getElementById('llmStatus');
        const dataStatus = document.getElementById('dataStatus');
        const exampleQueries = document.querySelectorAll('.example-query');

        let dataLoaded = false;

        // Check initial status
        checkStatus();

        async function checkStatus() {
            try {
                const response = await fetch('/api/status');
                const data = await response.json();
                
                if (data.neo4j_connected) {
                    neo4jStatus.classList.add('connected');
                }
                
                if (data.llm_connected) {
                    llmStatus.classList.add('connected');
                }
                
                if (data.data_loaded) {
                    dataStatus.classList.add('connected');
                    dataLoaded = true;
                    enableChat();
                    loadButton.style.display = 'none'; // Hide load button if data is already loaded
                }
            } catch (error) {
                console.log('Status check failed:', error);
            }
        }

        async function loadData() {
            try {
                loadButton.disabled = true;
                loadButton.textContent = '🔄 Loading...';
                
                addMessage('system', 'Loading Marvel Comics data into Neo4j database...');
                
                const response = await fetch('/api/load-data', {
                    method: 'POST'
                });
                
                const data = await response.json();
                
                if (data.success) {
                    addMessage('system', '✅ Data loaded successfully! You can now ask questions about Marvel characters.');
                    dataLoaded = true;
                    enableChat();
                    dataStatus.classList.add('connected');
                } else {
                    addMessage('system', '❌ Failed to load data: ' + data.error);
                }
            } catch (error) {
                addMessage('system', '❌ Error loading data: ' + error.message);
            } finally {
                loadButton.disabled = false;
                loadButton.textContent = '📊 Load Data';
            }
        }

        function enableChat() {
            queryInput.disabled = false;
            sendButton.disabled = false;
            exampleQueries.forEach(query => {
                query.classList.remove('disabled');
            });
        }

        function setQuery(query) {
            if (!dataLoaded) {
                addMessage('system', '⚠️ Please load the data first before asking questions.');
                return;
            }
            queryInput.value = query;
            queryInput.focus();
        }

        function addMessage(role, content, cypher, results, error, naturalResponse) {
            const messageDiv = document.createElement('div');
            messageDiv.className = 'message ' + role;
            
            const header = document.createElement('div');
            header.className = 'message-header';
            
            if (role === 'user') {
                header.textContent = '👤 You';
            } else if (role === 'assistant') {
                header.textContent = '🤖 Assistant';
            } else if (role === 'system') {
                header.textContent = '⚙️ System';
            }
            
            const contentDiv = document.createElement('div');
            contentDiv.className = 'message-content';
            contentDiv.textContent = content;
            
            messageDiv.appendChild(header);
            messageDiv.appendChild(contentDiv);
            
            if (cypher) {
                const cypherDiv = document.createElement('div');
                cypherDiv.className = 'cypher-query';
                cypherDiv.textContent = cypher;
                messageDiv.appendChild(cypherDiv);
            }
            
            if (naturalResponse) {
                const responseDiv = document.createElement('div');
                responseDiv.className = 'natural-response';
                responseDiv.textContent = naturalResponse;
                messageDiv.appendChild(responseDiv);
            }
            
            if (error) {
                const errorDiv = document.createElement('div');
                errorDiv.className = 'error';
                errorDiv.textContent = error;
                messageDiv.appendChild(errorDiv);
            }
            
            if (results) {
                const toggleButton = document.createElement('button');
                toggleButton.className = 'toggle-results';
                toggleButton.textContent = '🔍 Show Raw Results';
                toggleButton.onclick = function() {
                    const resultsDiv = this.nextElementSibling;
                    if (resultsDiv.style.display === 'none' || resultsDiv.style.display === '') {
                        resultsDiv.style.display = 'block';
                        this.textContent = '🔍 Hide Raw Results';
                    } else {
                        resultsDiv.style.display = 'none';
                        this.textContent = '🔍 Show Raw Results';
                    }
                };
                messageDiv.appendChild(toggleButton);
                
                const resultsDiv = document.createElement('div');
                resultsDiv.className = 'results';
                resultsDiv.textContent = results;
                messageDiv.appendChild(resultsDiv);
            }
            
            chatMessages.appendChild(messageDiv);
            chatMessages.scrollTop = chatMessages.scrollHeight;
        }

        async function sendQuery(query) {
            try {
                sendButton.disabled = true;
                loading.style.display = 'block';
                
                const response = await fetch('/api/query', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                    },
                    body: JSON.stringify({ query: query })
                });
                
                const data = await response.json();
                
                if (data.error) {
                    addMessage('assistant', 'I encountered an error while processing your query.', data.cypher, null, data.error);
                } else {
                    addMessage('assistant', 'Here\'s what I found in the Marvel knowledge graph:', data.cypher, data.results, null, data.response);
                }
            } catch (error) {
                addMessage('assistant', 'Sorry, I encountered an error while processing your request.', null, null, error.message);
            } finally {
                sendButton.disabled = false;
                loading.style.display = 'none';
            }
        }

        queryForm.addEventListener('submit', async function(e) {
            e.preventDefault();
            const query = queryInput.value.trim();
            if (!query || !dataLoaded) return;
            
            addMessage('user', query);
            queryInput.value = '';
            
            await sendQuery(query);
        });

        queryInput.addEventListener('input', function() {
            this.style.height = 'auto';
            this.style.height = Math.min(this.scrollHeight, 120) + 'px';
        });

        queryInput.addEventListener('keydown', function(e) {
            if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault();
                queryForm.dispatchEvent(new Event('submit'));
            }
        });
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Generate Cypher query using LLM
	cypherQuery, err := generateCypherQuery(llm, req.Query, schema)
	if err != nil {
		response := QueryResponse{
			Query:     req.Query,
			Cypher:    "",
			Results:   "",
			Error:     fmt.Sprintf("Failed to generate query: %v", err),
			Timestamp: getCurrentTimestamp(),
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Execute query and get results
	results := executeQuery(driver, cypherQuery)

	// Generate natural language response
	naturalResponse := generateNaturalResponse(llm, req.Query, cypherQuery, results)

	response := QueryResponse{
		Query:     req.Query,
		Cypher:    cypherQuery,
		Results:   results,
		Response:  naturalResponse,
		Timestamp: getCurrentTimestamp(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	response := map[string]bool{
		"neo4j_connected": driver != nil,
		"llm_connected":   llm != nil,
		"data_loaded":     dataLoaded,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleLoadData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Load data into Neo4j
	loadDataToNeo4j()

	dataLoaded = true
	response := map[string]interface{}{
		"success": true,
		"message": "Data loaded successfully",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func checkIfDataExists(driver neo4j.Driver) bool {
	session := driver.NewSession(neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close()

	// Check if there are any Character nodes
	result, err := session.Run("MATCH (c:Character) RETURN count(c) as count LIMIT 1", nil)
	if err != nil {
		return false
	}

	if result.Next() {
		record := result.Record()
		if len(record.Values) > 0 {
			count := record.Values[0].(int64)
			return count > 0
		}
	}

	return false
}

func generateNaturalResponse(llm llms.Model, userQuery, cypherQuery, results string) string {
	prompt := fmt.Sprintf(`You are a helpful assistant that explains Marvel Comics knowledge graph results in natural language.

User Question: "%s"
Cypher Query Executed: %s
Graph Database Results: %s

Generate a natural, conversational response that:
1. Directly answers the user's question
2. Explains the results in a friendly, engaging way
3. Highlights key relationships and connections
4. Uses Marvel Comics terminology appropriately
5. Keeps the response concise but informative
6. If no results found, explain what the user might try instead

Write a natural response as if you're a knowledgeable Marvel Comics expert:`, userQuery, cypherQuery, results)

	ctx := context.Background()
	response, err := llm.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	})
	if err != nil {
		return fmt.Sprintf("I found some information in the Marvel knowledge graph, but I couldn't generate a natural response. Here are the raw results: %s", results)
	}

	if len(response.Choices) == 0 {
		return fmt.Sprintf("I found some information in the Marvel knowledge graph, but I couldn't generate a natural response. Here are the raw results: %s", results)
	}

	return strings.TrimSpace(response.Choices[0].Content)
}

func getCurrentTimestamp() string {
	return fmt.Sprintf("%d", time.Now().Unix())
}
