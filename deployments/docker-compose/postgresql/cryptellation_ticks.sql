CREATE USER cryptellation_ticks;
ALTER USER cryptellation_ticks PASSWORD 'cryptellation_ticks';
ALTER USER cryptellation_ticks CREATEDB;

CREATE DATABASE cryptellation_ticks;
GRANT ALL PRIVILEGES ON DATABASE cryptellation_ticks TO cryptellation_ticks;
\c cryptellation_ticks postgres
GRANT ALL ON SCHEMA public TO cryptellation_ticks;