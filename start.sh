#!/usr/bin/env bash
if [ "$1" == "" ]
then
  echo "usage: $0 dbfile"
else
  dbfile=$1
  if [ ! -f "$dbfile" ]
  then
    echo '.quit' | sqlite3 --init observations.sql $dbfile
  else
    echo "db file exists, will not modify: $dbfile"
  fi

  if [ "$CONFIGPATH" == "" ]
  then
    echo "CONFIGPATH environment variable must be set"
  else
    if [ ! -f "$CONFIGPATH" ]
    then
      cp configexample.json $CONFIGPATH
      sed -i -e "s/xxxx.db/$dbfile/g" $CONFIGPATH
      vi $CONFIGPATH
    else
      echo "config file exists, will not modify: $CONFIGPATH"
    fi
  fi
fi