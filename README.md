# PAXOS-Banking
CS270-Advanced Distributed Systems Final Project


- Server will ALWAYS know the associated clients txns
- 
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
- Each server maintains a local blockchain: On addition of a block, this newly updated blockchain is stored in 
a persistent store as well (possibly redis?)

1. A server is identified to be a zombie when it comes up and does a store lookup. IF a blockchain is found
in the store for this particular server, it means that the server was up at one point in time - and thus, it is 
a zombie process. Else, the server is a baby.
2. In case of a zombie server, firstly, the blockchain is fetched from the store. Then, a special message is sent 
to each of the other servers, requesting "re-conciliation". <RECONCILE_REQUEST> 
3. Each of the other servers sends a list of sequence numbers contained in their respective blockchains. <R_ACK, [1,2,3]> 
4. The zombie servers picks the longest list and sends a message back to it asking for the blocks. Once it recieves
the blocks, it updates its own block chain and also caches it. Zombie: <R_BLOCKS_REQ, [2, 3]> ..... Response: <R_BLOCKS_RESP, [Ba, Bb]>

