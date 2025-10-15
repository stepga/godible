# go(au)dible

:-)

## Debugging/Infos

* Kernel info
  * `uname -r`
  * config `gzip -d /proc/config.gz -c  | less`
  * kernel parameters: `cat /proc/cmdline`

## TODOs

* add reading commands via unix socket for debugging
* rename AudioSource to Track ... or something less rough
* implement Previous()
* implement recursive file/dir watch and update Player.audioSourceList
  * e.g via https://github.com/fsnotify/fsnotify/issues/18#issuecomment-3109424560
* gpio: detect long press and implement other functions
  * e.g. fast forward

* support mp3/ogg
  * transform to wav when upload, OR
  * encode to wav during play
* web control interface
  * play/pause songs
  * save player state on /perm to survive reboots
  * upload songs
  * delete songs
* webcam qr code module
  * decide: via button push or e.g. one webcam shot per second check?
  * see also https://github.com/makiuchi-d/gozxing
* fritzing of the hardware setup
