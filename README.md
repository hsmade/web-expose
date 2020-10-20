# web-expose
Expose a remote webserver over websocket

The server is run on a public IP. It accepts a websocket connection from the client. Any web request done to the server is forwarded over the websocket, to the client, where the client will proxy the request to a local webserver, take the response and return it over the websocket, back to the server.
This way, you can expose a local webserver on a public IP, a la `ngrok`.
