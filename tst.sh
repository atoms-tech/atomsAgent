psql "postgresql://postgres:qkuTqRJkVypJleGb@db.ydogoylwenufckscqijp.supabase.co:5432/postgres" << EOF                                                    
    ALTER TABLE agents OWNER TO postgres;                                    
    ALTER TABLE models OWNER TO postgres;                                     
    ALTER TABLE chat_sessions OWNER TO postgres;                              
    ALTER TABLE chat_messages OWNER TO postgres;                              
    ALTER TABLE agent_health OWNER TO postgres;                               
                                                                              
    SELECT tablename, tableowner                                              
    FROM pg_tables                                                            
    WHERE schemaname = 'public'                                               
    AND tablename IN ('agents', 'models', 'chat_sessions', 'chat_messages',   
  'agent_health')                                                             
    ORDER BY tablename;                                                       
    EOF   
