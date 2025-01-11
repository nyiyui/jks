'use strict';

const handleKey = (e) => {
  if (e.target.matches('input, button, textarea')) {
    return;
  }
  if (e.key === 'L') {
    window.location.href = '/activity/latest';
  }
  if (e.key === '-') {
    window.location.href = '/day/yesterday';
  }
  if (e.key === 'D') {
    window.location.href = '/day/today';
  }
  if (e.key === '+') {
    window.location.href = '/day/tomorrow';
  }
  if (e.key === 'U') {
    window.location.href = '/undone-tasks';
  }
  if (e.key === 'T') {
    window.location.href = '/task/new';
  }
  if (e.key === 'A') {
    window.location.href = '/task/new/activity/new';
  }
  if (e.key === 'S') {
    window.location.href = '/login/settings';
  }
  console.log(e.key);
}

document.addEventListener('keyup', handleKey);
