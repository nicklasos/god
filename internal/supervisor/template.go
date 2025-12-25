package supervisor

// DefaultTemplate is the default template for creating new supervisord processes
const DefaultTemplate = `[program:{{name}}]
command={{command}}
directory={{directory}}
user={{user}}
autostart=true
autorestart=true
startsecs=10
startretries=3
stdout_logfile={{stdout_logfile}}
stderr_logfile={{stderr_logfile}}
stdout_logfile_maxbytes=1MB
stdout_logfile_backups=10
stderr_logfile_maxbytes=1MB
stderr_logfile_backups=10
environment={{environment}}
priority=999
stopsignal=TERM
stopwaitsecs=30
`
