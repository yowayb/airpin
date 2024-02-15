Airpin is a Google Cloud Function that copies Pinterest analytics to Airtable,
because Pinterest's reporting interface is frustrating.


#### Design

Metric/column names vary between Pinterest's interfaces (API v4, API v5, CSV, 
web UI) and there's no consistent mapping, so we hard-code the mapping.  This
is ok long-term because the names don't change.

We prefer to panic instead of handling errors because Airpin is only used
internally, and we absolutely don't want bad data making it to Airtable.
Debugging is done with print statements.  When not in use, leave these lines
commented.  Do not remove them, because you will need them later.

Airpin only appends rows to the Ad KPIs table in each base.  We have a base for
each client because Airtable's web UI encourages tables to be used like 
spreadsheets (rather than normalized tables).  

The selection of metrics is defined in a Pinterest report template.  There is
a v5 endpoint to request a report using the template, but it does not honor
the start and end dates, so we would have to make a template for each time range
and write brittle code to find them.  Instead, we manually update `config.json`
to match the template.  We rely on the Pinterest report template as the source
of truth of the report configuration.

Since the aforementioned mapping is hard-coded, whenever this selection changes,
we have to manually update the mapping.  This is facilitated by the
`./templates.sh` utility script which downloads a JSON of all templates from the
Pinterest account.  We then copy the columns from the JSON and paste them into
`config.json` and make the necessary changes to `airtable.go`.

The fields of each Ad KPIs table correspond to the columns in the Pinterest 
report template, but we use abbreviations because the Pinterest display names
are long, making it cumbersome to navigate when there's a lot of metrics.   


#### Requirements

- Google Cloud CLI installed and authenticated
- In Google Cloud Secret Manager:
  - `airtable-token`
  - `b64client`
  - `pinterest-refresh-token`
  - `pinterest-token`


#### Setup

Deploy `oauth.go` and schedule it to run when the Pinterest token expires.  This
will refresh the token automatically when it expires.


#### Changing, Testing and Deploying

Read the contents of the shell scripts.  The scripts are named after what they
do.  You MUST run `source creds.sh` before any other scripts will work.
