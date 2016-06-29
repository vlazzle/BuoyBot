# this is for local development, since heroku will have this set automatically
if [ "$DATABASE_URL" == "" ]
then
  # for this environment variable to be set in the shell running this script, this script should be run with source, i.e. `source start.sh`
  export DATABASE_URL=postgres:///$(whoami)?sslmode=disable
fi

# to create tables
psql $DATABASE_URL -f observations.sql