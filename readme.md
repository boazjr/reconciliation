# Play

TODO: 

[] after connection client change velocity and guesses the object position
[] server accepts the new velocity
[] client predicts the objects new position.

edge cases

client moving velocity or distance> sends information to server > 1/10 gets dropped.
server is aware of 9/10 movements, client is aware 10/10

expectations: 
c: > > > > > > d > > > x
s: > > > > > > > > x


Velocity - velocity of the object - ignore this implementation because there are more use cases for the ConstantVelocity.

ConstantVelocity - which doesn't change - or performs a single operation. make sure that client sends velocity multiple times with the same server cycle.

list of the last x changes the client made, and which server cycle they made them
send the entire list for y number of times.

server needs to be able to store the history - a way to compensate for lag.


sc - server cycle
v - velocity
p - position

client
sc v  p 
1  1  0 - client performed change in velocity after the update
2  1  0+1 - change in velocity at sc 
3  1  1+1
4  1  3
5  1  4
6  1  5

server
sc  v p
1   0 0
2   1 0
3   1 1 - server sends the p at sc

3,1 - position update of the server


reconcile
client
sc  v  p
1   1  0 - client performed change in velocity after the update
2   1  0+1 - change in velocity at sc 
3   1  1+1 - reconcile so that 


client updates to server
sc,v - multiples 

server updates to client
sc,p - most recent position
