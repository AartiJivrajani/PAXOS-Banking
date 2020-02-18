# PAXOS-Banking
CS270-Advanced Distributed Systems Final Project

### Deployment

To run redis locally, make sure you have the `redis.conf` file. 
Run the below command to start the redis server in the background
```bash
redis-server ./redis.conf
```

Questions - 
1. When the server fails, and it had some pending local txns, what happens whjen the server comes back up? 
    - Should it have persisted its own block chain and the local txns? 


#### Design Discussion - 1(15th Feb)

For the sake of brevity, we thought it would be funny and convinient to coin a term for 2 types of processes - 
1. Baby server: A server which comes up for the very first time. 
2. Zombie server: A server which had originally crashed and came up post the crash. 

System configuration: 
- 3 servers(A, B, C)
- 3 clients(A, B, C) - associated with each server
- Each server maintains a local block chain: On addition of a block, this newly updated block chain is stored in 
a persistent store as well (possibly redis?)

1. A server is identified to be a zombie when it comes up and does a store lookup. IF a blockchain is found
in the store for this particular server, it means that the server was up at one point in time - and thus, it is 
a zombie process. Else, the server is a baby.
2. In case of a zombie server, firstly, the block chain is fetched from the store. Then, a special message is sent 
to each of the other servers, requesting "re-conciliation". `<RECONCILE_REQUEST>` 
3. Each of the other servers sends a list of sequence numbers contained in their respective blockchains. `<R_ACK, [1,2,3]>` 
4. The zombie servers picks the longest list and sends a message back to it asking for the blocks. Once it receives
the blocks, it updates its own block chain and also caches it. Zombie: `<R_BLOCKS_REQ, [2, 3]>` ..... Response: `<R_BLOCKS_RESP, [Ba, Bb]>`

TODO:   
[1] Write the client side code.   
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; - Send a request to the server.   
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; - Keep a timeout for this request. If this times out, try again after a while(backoff and retry).   
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; - Process the response received from the server  
 
[2] Server side implementation  
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; [a] Establish the n/w topology    
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; &nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; - Maintain sticky connections with each of the other servers.    
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; - Open a socket and listen for any server(PAXOS runs)/client(requests) events    
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; - Open a connection to REDIS and keep the connection sticky.    
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; [b] Take care of 2 cases on startup  
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; - Zombie server process  
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; - lookup redis data, send out requests to each of the servers, receive list of seq numbers from each, follow up
          with requests for specific block numbers. Once received, cache these blocks and add them to the block chain  
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; - Baby server process  
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; - When the server comes up, check redis, if no cached data found, don't do anything after establishing the network topology  
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; [c] Process client transactions  
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; - Check the local balance, if amount sufficient, prepare a response and send back to the client. Add the current transaction
        to the local log.  
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; - If the local amount is insufficient to process the transaction, start a PAXOS run.   

#### REDIS DATA MODEL
Key: SERVER-BLOCKCHAIN-<id> Value: the committed blockchain
Key: SERVER-LOG-<id> Value: uncommitted log

