README

# Server

### High Level Summary
The server's job is esssentially to manage connections with clients, streams data from each station, and handle user input. 

Each client is associated with a unique number when it connects to the server. We store client information in a map from this unique number to a struct containing its connection with the server, the udp connection where it is listening, the udp port, and the station it is listening to. 

We have another map from each station to the clients that are listening to it. This is updated according to information from the client. 

These two maps are each protected by a mutex to avoid race conditions. 

In terms of concurrency, we have a (1) thread for each station that runs regardless of if clients are listening, (2) a thread for accepting new connections from clients, and (3) a thread for each client that has connected.  

### By Function
Main takes in the appropriate arguments for the server. It will process them, setting up a listening address for clients to connect. It then initializes the two maps and creates a thread to (2) wait_for_connections. We then create a thread for each station to (1) stream. Finally, we wait indefinitely for user input and process it appropriately. 

(1) stream takes in the file that the station data is stored in and the index of the station. It will indefinitely cycle through this file, storing the information in a buffer and jumping back to the beginning of the file when the end is reached. Each time we fill the buffer with information, we will check our maps for the clients that should be receiving data. We send it, and sleep the thread for a dynamic amount of time to keep the stream rate at roughly 16 kB/s. 

(2) wait_for_connections will accept incoming connections and delegate the task of handling them to (3) handle_Conn

(3) handle_Conn takes in a pointer to a tcpConn. To enforce timeout specifications, we set a deadline from the start of the method. We instantiate the struct for the client, with placeholders and the tcp connection. We associate this client with its unique number, update the unique number that will be assigned to the next client that joins, and add the client's info to the map. We now wait indefinitely for client messages. To allow messages to be sent in pieces, we continue reading into the appropriate index of our buffer until it is full, enforcing timeouts with a deadline. Once our message is fully read, we follow the program specifications for responding. 

(4) announce is used when a client sets a station and when a station is restarting the song. It takes a client num and station num as an argument. It first updates the internal maps-if the song being set is different from the client's current song, or if the client has not started listening yet, it will update the maps accordingly. It then sends the appropriate message to the client with its tcp connection. 

(5) invalid_command is used when the server receives a command that does not fit the program specifications (eg. multiple hellos, set station before hello, invalid set station number, etc.) It writes a message to the client with a message type 4, indicating that there was an error and to shut down. 

(6) clean removes clients from the maps when they disconnect for any reason. 

# Control

## High Level Summary 
The client communicates with the server to send data to the listener. We use several booleans to keep track of the state of communications. We indefinitely wait for messages from the server in main, and use a thread to wait for user input. 

### By Function
Main takes in the specified arguments. It sets up a connection with the server and initializes the states. It will set deadlines at appropriate steps to ensure that we do not wait too long for responses from the server. It sends the hello message to the server, and then begins the indefinite loop waiting for server messages. The loop will read the first byte from the server. If we know the incoming message is a welcome, we have not yet received a welcome, and we sent the hello, we will read in the number of stations piece by piece until the whole message is received. We then execute the specified behavior and start the thread to wait for user input (1). If the incoming message is an announce and we have set the station via user input, we first read the size of the song name, then read the song name piece by piece into an appropriately sized buffer and print it out. If the message does not meet either condition, we end the connection. 

(1) wait_for_input indefinitely scans for user input. If the user types q or a valid station number, it will quit or send the set station message to the server. Otherwise it will tell to user to do one of the two and restart the loop. 

end_connection is use whenever the client shuts down to send a message to the server, telling it to clean all the client information and kill the connection. 

