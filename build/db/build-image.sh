#!/bin/bash

rm -rf database
mkdir database
cp -r ../../wiki/db/monitor_sql_01_struct.sql  database/

if [ $1 == 'en' ]
then
cp -r ../../wiki/db/monitor_sql_02_base_data_en.sql  database/
else
cp -r ../../wiki/db/monitor_sql_02_base_data_cn.sql  database/
fi

cd database
for i in `ls -1 ./*.sql`; do
	sed -i '1 i\use monitor;SET NAMES utf8;' $i
done
cd ../

echo -e "SET NAMES utf8;\n" > ./database/000000_create_database.sql
echo -e "create database monitor charset = utf8;\n" >> ./database/000000_create_database.sql


docker build -t monitor-db:dev .
