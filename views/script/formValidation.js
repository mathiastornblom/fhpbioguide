document.addEventListener("DOMContentLoaded", function () {
	const form = document.querySelector("form");
	form.addEventListener("submit", function (event) {
		let isValid = true;
		// Example validation: Ensure no negative numbers
		form.querySelectorAll('input[type="number"]').forEach(function (input) {
			if (input.value < 0) {
				isValid = false;
				input.style.borderColor = "red"; // Highlight in red
			} else {
				input.style.borderColor = ""; // Reset
			}
		});

		if (!isValid) {
			event.preventDefault(); // Stop form submission
			alert(
				"Vänligen rätta till felen i formuläret. Fälten får inte vara negativa."
			);
		}
	});
});
