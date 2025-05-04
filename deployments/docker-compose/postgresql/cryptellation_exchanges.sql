CREATE USER cryptellation_exchanges;
ALTER USER cryptellation_exchanges PASSWORD 'cryptellation_exchanges';
ALTER USER cryptellation_exchanges CREATEDB;

CREATE DATABASE cryptellation_exchanges;
GRANT ALL PRIVILEGES ON DATABASE cryptellation_exchanges TO cryptellation_exchanges;
\c cryptellation_exchanges postgres
GRANT ALL ON SCHEMA public TO cryptellation_exchanges;