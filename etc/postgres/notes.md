
# Get stats on client connections
select datname,pid,client_addr,client_port,backend_start,query_start,state,waiting from pg_stat_activity ;

