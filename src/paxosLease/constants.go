package paxosLease

//max propose count is 2^(64-8-16) = 2^40 ~= (34865 year if +1/sec)
const PROPOSE_ID_WIDTH_RESTART_COUNTER = 16 //max restart count = 2^16 = 65534
const PROPOSE_ID_WIDTH_NODEID = 8           //max node count = 2^8-1 = 255
const PREPARING_TIMEOUT = 3000              //ms
const MAX_LEASED_TIME = 10000               //ms
