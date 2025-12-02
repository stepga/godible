var time_current_lock = false;

var is_playing = false;

// FIXME: double book keeping: assets/img/... & these hardcoded strings
// TODO: load the file content from the files, as in
//   - https://forum.freecodecamp.org/t/load-local-text-file-with-js/83063/7
//   - https://developer.mozilla.org/en-US/docs/Web/API/Fetch_API/Using_Fetch
const play_html = `
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
<path d="M371.7 238l-176-107c-15.8-8.8-35.7 2.5-35.7 21v208c0 18.4 19.8 29.8 35.7 21l176-101c16.4-9.1 16.4-32.8 0-42zM504 256C504 119 393 8 256 8S8 119 8 256s111 248 248 248 248-111 248-248zm-448 0c0-110.5 89.5-200 200-200s200 89.5 200 200-89.5 200-200 200S56 366.5 56 256z"/>
</path>
</svg>
<!--
Font Awesome Free 5.2.0 by @fontawesome - https://fontawesome.com
License - https://fontawesome.com/license (Icons: CC BY 4.0, Fonts: SIL OFL 1.1, Code: MIT License)
-->
`;
const pause_html = `
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512">
<path d="M256 8C119 8 8 119 8 256s111 248 248 248 248-111 248-248S393 8 256 8zm0 448c-110.5 0-200-89.5-200-200S145.5 56 256 56s200 89.5 200 200-89.5 200-200 200zm96-280v160c0 8.8-7.2 16-16 16h-48c-8.8 0-16-7.2-16-16V176c0-8.8 7.2-16 16-16h48c8.8 0 16 7.2 16 16zm-112 0v160c0 8.8-7.2 16-16 16h-48c-8.8 0-16-7.2-16-16V176c0-8.8 7.2-16 16-16h48c8.8 0 16 7.2 16 16z"/>
</path>
</svg>
<!--
Font Awesome Free 5.2.0 by @fontawesome - https://fontawesome.com
License - https://fontawesome.com/license (Icons: CC BY 4.0, Fonts: SIL OFL 1.1, Code: MIT License)
-->
`;

function secondsToDateStr(seconds) {
	var date = new Date(null);
	date.setSeconds(seconds);
	if (seconds < 60*60) {
		return date.toISOString().slice(14, 19);
	}
	return date.toISOString().slice(11, 19);
}

function dateStrToSeconds(date) {
    var arr = date.split(':');
    var ret = 0;
    var seconds_multiplier = 1;

    while (arr.length > 0) {
        ret += seconds_multiplier * parseInt(arr.pop(), 10);
        seconds_multiplier *= 60;
    }

    return ret;
}

function setToggleButton() {
	var toggle = document.getElementById("toggle");
	if (is_playing) {
		toggle.innerHTML = pause_html;
	} else {
		toggle.innerHTML = play_html;
	}
}

function updateUI(data) {
	if (data == null || data == "null") {
		console.log("updateUI: no data passed");
		return;
	}
	var json = JSON.parse(data);
	var title = document.getElementById("title");
	var time_current = document.getElementById("time_current");
	var time_total = document.getElementById("time_total");
	var slider = document.getElementById("slider");

	// TODO: update fields only if content really changed
	title.textContent = json.name;
	time_total.textContent = secondsToDateStr(json.duration);
	slider.max = json.duration;
	if (time_current_lock == false) {
		time_current.textContent = secondsToDateStr(json.duration_current);
		slider.value = json.duration_current;
	}

	is_playing = json.is_playing;
	setToggleButton();
}

document.addEventListener("DOMContentLoaded", function(event) {
	var websocket = new WebSocket("ws://"+window.location.host+"/ws");
	websocket.onmessage = function(event) {
		updateUI(event.data);
	}
	websocket.onerror = function(event) {
		console.log("WebSocket error: " + event.data);
	}

	var slider = document.getElementById("slider");
	slider.oninput = function() {
		const time_current = document.getElementById("time_current");
		time_current.textContent = secondsToDateStr(slider.value);
	}

	function downEvent() {
		time_current_lock = true;
	}
	slider.onmousedown = downEvent
	slider.ontouchstart = downEvent

	function upEvent() {
		const time_current_date_str = document.getElementById("time_current").textContent;
		const time_current_seconds = dateStrToSeconds(time_current_date_str);
		time_current_lock = false;
		websocket.send('{ "command": "jump", "payload": "' + time_current_seconds + '"}');
	}
	slider.onmouseup = upEvent
	slider.ontouchend = upEvent

	document.getElementById("previous").addEventListener('click', function() {
		websocket.send('{ "command": "previous", "payload": ""}');
	});
	document.getElementById("toggle").addEventListener('click', function() {
		websocket.send('{ "command": "toggle", "payload": ""}');
		is_playing = !is_playing;
		setToggleButton();
	});
	document.getElementById("next").addEventListener('click', function() {
		websocket.send('{ "command": "next", "payload": ""}');
	});
});
