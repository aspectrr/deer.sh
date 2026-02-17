-- Create the fluid_web database for the web API service
-- The default 'fluid' database is used by fluid-remote
SELECT 'CREATE DATABASE fluid_web OWNER fluid'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'fluid_web')\gexec
