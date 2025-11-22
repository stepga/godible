function click_action() {
	//if (!newpageText.value || newpageText.value.length === 0) {
	//	console.log("input action: nothing to do, value is missing")
	//	return;
	//}
	getData()
}

async function getData() {
	const url = "http://localhost:1234/state";
	try {
		const response = await fetch(url);
		if (!response.ok) {
			throw new Error(`Response status: ${response.status}`);
		}

		const result = await response.json();
		console.log(result);
	} catch (error) {
		console.error(error.message);
	}
}

document.addEventListener("DOMContentLoaded", function(event) {
	var slider = document.getElementById("myRange");
	var output = document.getElementById("demo");
	output.innerHTML = slider.value;
	slider.oninput = function() {
		output.innerHTML = this.value;
	}

	const newpageButton = document.querySelector('#input-button');
	const newpageText = document.querySelector('#input-text');
	newpageText.addEventListener("keydown", function(event) {
		if (event.keyCode === 13) { // 13 == 0x0d == CR (ASCII) == Enter
			event.preventDefault();
			newpageButton.click();
		}
	});
	newpageButton.addEventListener('click', click_action);
});

window.addEventListener("load", function(evt) {
	var output = document.getElementById("output");
	var input = document.getElementById("input");
	var ws;

	var print = function(message) {
		var d = document.createElement("div");
		d.textContent = message;
		output.appendChild(d);
		output.scroll(0, output.scrollHeight);
	};

	document.getElementById("open").onclick = function(evt) {
		if (ws) {
			return false;
		}
		ws = new WebSocket("ws://"+window.location.host+"/ws");
		ws.onopen = function(evt) {
			print("OPEN");
		}
		ws.onclose = function(evt) {
			print("CLOSE");
			ws = null;
		}
		ws.onmessage = function(evt) {
			print("RESPONSE: " + evt.data);
		}
		ws.onerror = function(evt) {
			print("ERROR: " + evt.data);
		}
		return false;
	};

	document.getElementById("send").onclick = function(evt) {
		if (!ws) {
			return false;
		}
		print("SEND: " + input.value);
		ws.send(input.value);
		return false;
	};

	document.getElementById("close").onclick = function(evt) {
		if (!ws) {
			return false;
		}
		ws.close();
		return false;
	};
});
