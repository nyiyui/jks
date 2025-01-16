const chosenTimezone = document.getElementById('timezone').value;
const clientTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

if (chosenTimezone !== clientTimezone) {
  document.getElementById('timezone-browser').value = clientTimezone;
  document.getElementById('timezone-alert').style.display = 'block';
}
