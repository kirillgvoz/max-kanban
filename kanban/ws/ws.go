package ws

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(request *http.Request) bool {
		origin := request.Header.Get("Origin")
		if origin == "" {
			return true
		}
		parsed, err := url.Parse(origin)
		return err == nil && strings.EqualFold(parsed.Host, request.Host)
	},
}

func HandleWebSocket(c *gin.Context) {
	ticket, ok := DefaultHub.ConsumeTicket(c.Query("ticket"))
	if !ok {
		c.JSON(401, gin.H{"error": "invalid or expired websocket ticket"})
		return
	}

	connection, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	client := &Client{
		Hub:       DefaultHub,
		UserID:    ticket.UserID,
		OrgID:     ticket.OrgID,
		Send:      make(chan []byte, 256),
		CloseConn: func() { _ = connection.Close() },
	}
	DefaultHub.Register(client)
	logConnection("WebSocket connected", ticket.OrgID)
	go client.WritePump(connection)
	go client.ReadPump(connection)
}

func BroadcastToOrg(_ *gin.Context, orgID int64, messageType string, data interface{}) {
	BroadcastToOrgDirect(orgID, messageType, data)
}

func BroadcastToOrgDirect(orgID int64, messageType string, data interface{}) {
	DefaultHub.Broadcast(orgID, Message{Type: messageType, Data: data})
}
