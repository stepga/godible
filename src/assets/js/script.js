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
