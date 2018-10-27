#Usage of Peerster:
---
- **UIPort** string
    port for the UI client (default "8080")
    **The GUI is served at this same port, so on your browser you should enter localhost:8080**
---
- **gossipAddr** string
	ip:port for the gossiper (default "127.0.0.1:5000")
---
- **name** string
	Name of the gossiper
---
- **peers** string
	Comma separated list of peers of the form ip:port
---
- **simple**
	Run Gossiper in simple broadcast mode