#Usage of client:
---
- **UIPort** string
    port for the UI client (default "8080")
---
- **dest** string
	Destination for the private message (must be peer name)
---
- **file** string
	File to be indexed by the gossiper, or filename of the requested file
---
- **msg** string
	Message to be sent
---
- **request** string
	Request a chunk or a metafile of this hash
---
- **keywords** string
	Comma separated values that will be searched in the names of the files shared by peers
---
- **budget** number
	(Optional) Starting search budget (how many peers we will search the file on)
