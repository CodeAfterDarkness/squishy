console.log("app loaded");

function loadWO(id){
	$.ajax({
		url: "../wo/" + id,
		success: function(json){
			//console.log(json);

			$('#woID').val(json.ID);
			$('#woName').val(json.OrderName);
			$('#woSummary').val(json.Summary);
			$('#woDetails').val(json.Details);
			$('#woCustomerName').val(json.CustomerName);

			window.location.hash = json.ID;

		}
	})
}

function createWO(){

	data = {};
	data.OrderName = $('#woMgmtName').val();
	data.Summary = $('#woMgmtSummary').val();
	data.Details = $('#woMgmtDetails').val();
	data.CustomerName = $('#woMgmtCustomerName').val();

	$.ajax({
		data: JSON.stringify(data),
		method: "POST",
		url: "../wo",
		success: function(json){
			console.log(json);

		}
	})
}