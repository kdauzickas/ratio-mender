############# ADD DATA HOW TO MEND RATIO!

### Ratio mender 
... is an app that helps you improve your torrent ratio.

### Using
RM works as a proxy so you'll need to configure your client to use it. Basically you just need to set the http proxy address to `localhost` and to use the port you reserved for RM (`57998` by default).

[Deluge](http://s2.postimage.org/e4m8jbwtl/deluge.png) | [uTorrent](http://s2.postimage.org/ra1qpfqp5/utorrent.png)

```
Ratio mender 0.1.1 usage:
  ratiomender [options]

Options:
  -d=1: By how much should the download be multiplied
  -h=false: Print this help and exit
  -l=false: Print log entries to output
  -p=57998: Port to listen
  -s=false: Switch download with upload. Multipliers are applied before the switch
  -u=1: By how much should the upload be multiplied
```

You can access the log with last 100 entries of current session by going http://localhost:port/log, where port - the port you reserved for RM, e.g. http://localhost:57998/log

On linux you can somewhat simulate a daemon by running this command
```
nohup /path/to/ratiomender > /var/log/ratiomender.log &
```
Or adding the following command to you start up script (Ubuntu: gnome-session-properties)
```
/path/to/ratiomender > /var/log/ratiomender.log
```

### Building
On linux just run
```
go build ratiomender.go icon.go
```

On windows you might want to add a few build flags:
```
go build -ldflags -Hwindowsgui ratiomender.go icon.go
```
This will build the app to launch withought a console window. When built like this `-l` and `-h` flags do nothing.

**Disclaimer:** I take no responsibility of what might happen while you use this piece of software. You most likely *will get banned* if you're cought using it. Note that this project is my way of learning Go and it's probably not optimal and buggy. Basically, use it at your own risk.
