package main

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "html/template"
    "io"
    "log"
    "net/http"
    "os"
    "strings"
    "time"

    "github.com/gorilla/mux"
    "github.com/joho/godotenv"
    "golang.org/x/image/draw"
)

type Config struct {
    GroqAPIKey     string
    GeminiAPIKey   string
    AppPort        string
}

type ChatMessage struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ChatRequest struct {
    Messages []ChatMessage `json:"messages"`
    Model    string        `json:"model"`
    Stream   bool          `json:"stream"`
}

type ChatResponse struct {
    ID      string      `json:"id"`
    Object  string      `json:"object"`
    Created int64       `json:"created"`
    Model   string      `json:"model"`
    Choices []Choice    `json:"choices"`
}

type Choice struct {
    Index        int     `json:"index"`
    Message      Message `json:"message"`
    FinishReason string  `json:"finish_reason"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}

type ImageRequest struct {
    Prompt string `json:"prompt"`
}

type ImageResponse struct {
    Candidates []ImageCandidate `json:"candidates"`
}

type ImageCandidate struct {
    Content ImageContent `json:"content"`
}

type ImageContent struct {
    Parts []ImagePart `json:"parts"`
}

type ImagePart struct {
    InlineData InlineData `json:"inlineData"`
}

type InlineData struct {
    MimeType string `json:"mimeType"`
    Data     string `json:"data"`
}

var config Config
var chatHistory []ChatMessage

func main() {
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }

    // Initialize config
    config = Config{
        GroqAPIKey:   os.Getenv("GROQ_API_KEY"),
        GeminiAPIKey: os.Getenv("GEMINI_API_KEY"),
        AppPort:      os.Getenv("PORT"),
    }

    if config.GroqAPIKey == "" || config.GeminiAPIKey == "" {
        log.Fatal("Missing required API keys. Set GROQ_API_KEY and GEMINI_API_KEY")
    }

    if config.AppPort == "" {
        config.AppPort = "8080"
    }

    r := mux.NewRouter()
    
    // Static files
    r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
    
    // Routes
    r.HandleFunc("/", indexHandler).Methods("GET")
    r.HandleFunc("/chat", chatHandler).Methods("POST")
    r.HandleFunc("/image", imageHandler).Methods("POST")
    r.HandleFunc("/new-chat", newChatHandler).Methods("POST")
    r.HandleFunc("/rewrite", rewriteHandler).Methods("POST")

    fmt.Printf("🚀 ARISE AI launched on :%s\n", config.AppPort)
    log.Fatal(http.ListenAndServe(":"+config.AppPort, r))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
    tmpl := template.Must(template.ParseFiles("templates/index.html"))
    tmpl.Execute(w, struct {
        ChatHistory []ChatMessage
    }{ChatHistory: chatHistory})
}

func chatHandler(w http.ResponseWriter, r *http.Request) {
    if r.Body == nil {
        http.Error(w, "Empty request body", http.StatusBadRequest)
        return
    }

    body, _ := io.ReadAll(r.Body)
    var req ChatRequest
    json.Unmarshal(body, &req)

    // Add user message to history
    chatHistory = append(chatHistory, ChatMessage{Role: "user", Content: req.Messages[len(req.Messages)-1].Content})

    // Call Groq API
    client := &http.Client{Timeout: 30 * time.Second}
    groqReq := ChatRequest{
        Messages: chatHistory,
        Model:    "llama-3.3-70b-versatile",
        Stream:   false,
    }

    jsonData, _ := json.Marshal(groqReq)
    groqResp, err := client.Post("https://api.groq.com/openai/v1/chat/completions", 
        "application/json", bytes.NewBuffer(jsonData))
    
    if err != nil {
        http.Error(w, "Groq API error", http.StatusInternalServerError)
        return
    }
    defer groqResp.Body.Close()

    var response ChatResponse
    json.NewDecoder(groqResp.Body).Decode(&response)

    aiMessage := response.Choices[0].Message
    chatHistory = append(chatHistory, aiMessage)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(aiMessage)
}

func imageHandler(w http.ResponseWriter, r *http.Request) {
    body, _ := io.ReadAll(r.Body)
    var req ImageRequest
    json.Unmarshal(body, &req)

    client := &http.Client{Timeout: 60 * time.Second}
    imageReq := fmt.Sprintf(`{
        "contents": [{
            "parts": [{
                "text": "Generate a high-quality image of: %s"
            }]
        }],
        "generationConfig": {
            "response_mime_type": "image/png",
            "response_modalitiy": "PRODUCE_IMAGES"
        }
    }`, req.Prompt)

    imageResp, err := client.Post("https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-exp:generateContent?key="+config.GeminiAPIKey,
        "application/json", strings.NewReader(imageReq))
    
    if err != nil {
        http.Error(w, "Gemini API error", http.StatusInternalServerError)
        return
    }
    defer imageResp.Body.Close()

    var imgResp ImageResponse
    json.NewDecoder(imageResp.Body).Decode(&imgResp)

    if len(imgResp.Candidates) > 0 && len(imgResp.Candidates[0].Content.Parts) > 0 {
        imgData := imgResp.Candidates[0].Content.Parts[0].InlineData
        w.Header().Set("Content-Type", imgData.MimeType)
        w.Header().Set("Content-Disposition", "inline; filename=arise-image.png")
        data, _ := base64.StdEncoding.DecodeString(imgData.Data)
        w.Write(data)
    } else {
        http.Error(w, "No image generated", http.StatusInternalServerError)
    }
}

func newChatHandler(w http.ResponseWriter, r *http.Request) {
    chatHistory = []ChatMessage{}
    w.WriteHeader(http.StatusOK)
}

func rewriteHandler(w http.ResponseWriter, r *http.Request) {
    if len(chatHistory) < 2 {
        http.Error(w, "No message to rewrite", http.StatusBadRequest)
        return
    }

    // Get last user message
    lastUserMsg := chatHistory[len(chatHistory)-2].Content
    rewritePrompt := fmt.Sprintf("Rewrite this message to make it better, more clear, and more effective: %s", lastUserMsg)

    // Add rewrite request to history
    chatHistory = append(chatHistory[:len(chatHistory)-1], ChatMessage{Role: "user", Content: rewritePrompt})

    // Call Groq for rewrite
    client := &http.Client{Timeout: 30 * time.Second}
    reqBody := ChatRequest{
        Messages: chatHistory,
        Model:    "llama-3.3-70b-versatile",
    }

    jsonData, _ := json.Marshal(reqBody)
    resp, err := client.Post("https://api.groq.com/openai/v1/chat/completions",
        "application/json", bytes.NewBuffer(jsonData))
    
    if err != nil {
        http.Error(w, "Rewrite failed", http.StatusInternalServerError)
        return
    }
    defer resp.Body.Close()

    var response ChatResponse
    json.NewDecoder(resp.Body).Decode(&response)
    
    rewrittenMsg := response.Choices[0].Message.Content
    chatHistory[len(chatHistory)-1] = ChatMessage{Role: "user", Content: rewrittenMsg}
    
    aiResponse := response.Choices[0].Message
    chatHistory = append(chatHistory, aiResponse)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"rewritten": rewrittenMsg, "response": aiResponse.Content})
}
