/* formValidation.js — FHP Report Forms */

document.addEventListener("DOMContentLoaded", function () {
  /* --- Sold form: dynamic discount rows + minimum price warning --- */
  document.querySelectorAll(".event-card").forEach(function (card) {
    var rabatterStr = card.dataset.rabatter || "";
    var minPrice = parseFloat(card.dataset.minprice) || 0;

    var rabatter = rabatterStr
      ? rabatterStr.split(",").map(function (s) { return parseInt(s.trim(), 10); })
      : [];

    /* Show only discount rows whose code is listed on the booking */
    card.querySelectorAll(".discount-row").forEach(function (row) {
      var kod = parseInt(row.dataset.rabattKod, 10);
      if (!rabatter.includes(kod)) {
        row.style.display = "none";
      }
    });

    /* Minimum price: pre-fill and warn if below */
    if (minPrice > 0) {
      var priceInput = card.querySelector(".ordinarie-price");
      var warning = card.querySelector(".price-warning");
      if (priceInput && warning) {
        priceInput.value = minPrice;
        priceInput.addEventListener("input", function () {
          var val = parseFloat(this.value);
          var below = !isNaN(val) && val > 0 && val < minPrice;
          warning.classList.toggle("price-warning--hidden", !below);
          priceInput.classList.toggle("input-warning", below);
        });
      }
    }
  });

  var form = document.getElementById("report-form");
  if (!form) return;

  /* Remove error highlight as soon as the user corrects a field */
  form.addEventListener("input", function (event) {
    var target = event.target;
    if (target.type === "number" || target.type === "url") {
      target.classList.remove("input-error");
    }
  });

  form.addEventListener("submit", function (event) {
    var inputs = form.querySelectorAll('input[type="number"]');
    var isValid = true;

    inputs.forEach(function (input) {
      var value = parseFloat(input.value);
      if (!isNaN(value) && value < 0) {
        input.classList.add("input-error");
        isValid = false;
      } else {
        input.classList.remove("input-error");
      }
    });

    if (!isValid) {
      event.preventDefault();
      alert("Vänligen rätta till felen i formuläret. Fälten får inte vara negativa.");
      /* Scroll to the first invalid field */
      var firstError = form.querySelector(".input-error");
      if (firstError) {
        firstError.scrollIntoView({ behavior: "smooth", block: "center" });
        firstError.focus();
      }
      return;
    }

    /* Valid — show loading state on the submit button */
    var submitBtn = document.getElementById("submit-btn");
    if (submitBtn) {
      submitBtn.classList.add("loading");
      submitBtn.disabled = true;
    }
  });
});
