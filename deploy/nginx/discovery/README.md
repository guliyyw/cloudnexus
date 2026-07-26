# Nginx discovery overrides

Files ending in `.conf` in this directory are included inside the nginx `server`
block after the default Docker service targets are defined.

Example for a later cross-server discovery controller:

```nginx
set $user_file_backend "http://10.0.0.11:8081";
set $im_backend "http://10.0.0.12:8082";
set $drama_backend "http://10.0.0.13:8087";
```

For the current single-host Docker deployment this directory can remain empty.
