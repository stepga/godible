var time_current_lock = false;

const pause_html = '<embed src="img/Font_Awesome_5_regular_pause-circle.svg">';
const play_html = '<embed src="img/Font_Awesome_5_regular_play-circle.svg">';

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

	var toggle = document.getElementById("toggle");
	if (json.is_playing == true) {
		if (toggle.innerHTML.toString() != pause_html) {
			toggle.innerHTML = pause_html;
		}
	} else {
		if (toggle.innerHTML.toString() != play_html) {
			toggle.innerHTML = play_html;
		}
	}
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
		console.log("disable automatic time_current updates due to slider action");
	}
	slider.onmousedown = downEvent
	slider.ontouchstart = downEvent

	function upEvent() {
		const time_current_date_str = document.getElementById("time_current").textContent;
		const time_current_seconds = dateStrToSeconds(time_current_date_str);
		time_current_lock = false;
		console.log("reenable automatic time_current updates");
		websocket.send('{ "command": "jump", "payload": "' + time_current_seconds + '"}');
	}
	slider.onmouseup = upEvent
	slider.ontouchend = upEvent

	document.getElementById("previous").addEventListener('click', function() {
		websocket.send('{ "command": "previous", "payload": ""}');
	});
	document.getElementById("toggle").addEventListener('click', function() {
		websocket.send('{ "command": "toggle", "payload": ""}');
	});
	document.getElementById("next").addEventListener('click', function() {
		websocket.send('{ "command": "next", "payload": ""}');
	});
});
