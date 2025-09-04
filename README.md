# go(au)dible

:-)

## Debugging/Infos

* Kernel info
  * `uname -r`
  * config `gzip -d /proc/config.gz -c  | less`
  * kernel parameters: `cat /proc/cmdline`

## TODOs

* play sound via soundcard
  * beatbox uses custom kernel and self-written minimal alsa (in go)
  * custom kernel does not seem necessary anymore
  (https://github.com/gokrazy/gokrazy/discussions/238#discussioncomment-7968624),
  but changing config.json & using the go-written alsa is probably still necessary
  (https://github.com/gokrazy/gokrazy/discussions/238#discussioncomment-7968624)
  * TODO: test if
  https://github.com/anisse/beatbox-kernel/commit/f597f35750fa1703a81bef621056225faaec3237
  is really within current gokrazy kernel
* music player module
* web control interface
  * play/pause songs on /perm (save state)
  * upload songs
  * delete songs
  * edit playlist
* webcam qr code module (via button push?)
* fritzing of the hardware setup
