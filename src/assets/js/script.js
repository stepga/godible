function registerFilterSearch() {
	$("#filterInput").on("keyup", function() {
		var filterSearchString = $(this).val().toLowerCase();
		$("tbody tr").filter(function() {
			// directory of iterated row matches the search:
			// show all corresponding rows
			var directoryRow = $(this).parent().find('tr').eq(0);
			var directoryRowText = directoryRow.text().toLowerCase();
			if (directoryRowText.indexOf(filterSearchString) > -1) {
				$(this).toggle(true);
				return;
			}

			// iterated row (may also be the directory row) matches the search:
			// show row and its corresponding directory row
			var row = $(this);
			var rowText = row.text().toLowerCase();
			if (rowText.indexOf(filterSearchString) > -1) {
				$(this).toggle(true);
				directoryRow.toggle(true);
			} else {
				$(this).toggle(false);
			}
		});
	});
}

function initializeWebsocket(websocket) {
	if (typeof(websocket) == 'undefined' || websocket == null) {
		console.log('initialize new websocket connection')
		websocket = new WebSocket("ws://"+window.location.host+"/ws");
	}
	websocket.onmessage = function(event) {
		let data = JSON.parse(event.data);
		switch (data['type']) {
			//TODO
			//case "state":
			//	updateUI(data['payload']);
			//	break;
			//case "rows":
			//	updateTable(data['payload']);
			//	break;
			//case "hiderfidalertbox":
			//	hideAlertBox(true, "");
			//	break;
			default:
				console.log("unknown websocket api request type: " + data['type'])
		}
	}
	websocket.onerror = function(event) {
		console.log("WebSocket error: " + event.data);
	}
}

$(document).ready(function(){
	registerFilterSearch();
	// TODO:
	// - slider
	// - table tbody hiding
	//
	//
	var websocket;
	initializeWebsocket(websocket);
});
