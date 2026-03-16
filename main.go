package main
import (
	"github.com/zeningc/mini-kafka/broker"
	"github.com/zeningc/mini-kafka/api"
	"net/http"
)

func main() {
	b := broker.NewBroker()

	server := api.NewServer(b)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /topics/{name}", server.HandleCreateTopic)
	mux.HandleFunc("POST /topics/{name}/messages", server.HandleProduce)
	mux.HandleFunc("GET /topics/{name}/messages", server.HandleConsume)

	http.ListenAndServe(":8080", mux)
}