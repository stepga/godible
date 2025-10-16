# go(au)dible

:-)

## Debugging/Infos

* Kernel info
  * `uname -r`
  * config `gzip -d /proc/config.gz -c  | less`
  * kernel parameters: `cat /proc/cmdline`

## FIXMEs

* Previous() does not work if there is a unsupported or broken file, as doPlay() immediately fails
  * ... live with this for now, OR
  * exclude the track from trackList if doPlay() fails n times?

## TODOs

* support mp3/ogg
  * transform to wav when upload, OR
  * encode to wav during play
* add reading commands via unix socket for debugging
* implement Previous() reset of currently played track
  * introduce a (percentage) threshold of the file being played
  * when this threshold is not yet reached, change the track
  * otherwise: reset the current track's offset.
* implement recursive file/dir watch and update Player.audioSourceList
  * e.g via https://github.com/fsnotify/fsnotify/issues/18#issuecomment-3109424560
* gpio: detect long press and implement other functions
  * e.g. fast forward, poweroff, etc.
  * counting time during WaitForEdge() does not work ... the straightforward attempt results in a broken gpio pin edge handling :-/
  * ... loop goroutines on the other hand are ugly ...
* web control interface
  * play/pause songs
  * save player state on /perm to survive reboots
  * upload songs
  * delete songs
* webcam qr code module
  * decide: via button push or e.g. one webcam shot per second check?
  * see also https://github.com/makiuchi-d/gozxing
* further circuitry stuff:
  * phone connector (klinkenstecker) via gpio (the raspberry pi zero 2w does not have a dedicated phone connector)
  * flip switch for power off/on
  * webcam via usb
* fritzing of the hardware setup
