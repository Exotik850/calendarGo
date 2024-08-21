<script lang="ts">
  import { onMount } from 'svelte';

  // Summary - ID 
  let calendars: Map<string, string> = new Map();
  // Summaries
  let selectedCalendars: string[] = [];
  let numDays = 7;
  let duration = 60; // minutes
  let eventLoc = '';
  let startLoc = '';

  // onMount(async () => {
  //   // Check if the cookie "authCodeEvPlanner" exists
    

  // });

  async function getCalendars() {
    const authCode = document.cookie
      .split('; ')
      .find(row => row.startsWith('authCodeEvPlanner'))
      ?.split('=')[1];

    if (!authCode) {
      // Redirect to the login page
      console.log('No auth code found. Redirecting to login page');
      // window.location.href = '/login';
      return;
    }

    // Fetch the list of calendars
    console.log('Fetching list of calendars...');
    const ids = await fetch('/listCalendars');
    console.log('Calendar IDs:', await ids.text());
  }

  function handleSubmit() {
    const formData = {
      calendars: selectedCalendars,
      numDays,
      duration,
      eventLoc,
      startLoc
    };
    
    // You'll handle the HTTP request to the Go server here
    console.log('Form data to send:', formData);
  }
</script>

<main>
  <h1>Event Scheduler</h1>

  <form on:submit|preventDefault={handleSubmit}>
    <div>
      <label>
        Select Calendars:
        <select multiple bind:value={selectedCalendars}>
          {#each calendars as key, summary}
            <option value={key}>{summary}</option>
          {/each}
        </select>
      </label>
    </div>

    <div>
      <label>
        Number of Days to Search:
        <input type="number" bind:value={numDays} min="1" max="30">
      </label>
    </div>

    <div>
      <label>
        Event Duration (minutes):
        <input type="number" bind:value={duration} min="15" step="15">
      </label>
    </div>

    <div>
      <label>
        Event Location:
        <input type="text" bind:value={eventLoc} placeholder="Enter address">
      </label>
    </div>

    <div>
      <label>
        Start Location:
        <input type="text" bind:value={startLoc} placeholder="Enter address">
      </label>
    </div>

    <button type="submit">Find Best Spot</button>
    <button on:mousedown={getCalendars} on:click|preventDefault>Get Calendars</button>
  </form>
</main>

<style>
  main {
    max-width: 600px;
    margin: 0 auto;
    padding: 20px;
  }

  form {
    display: flex;
    flex-direction: column;
    gap: 15px;
  }

  label {
    display: flex;
    flex-direction: column;
    gap: 5px;
  }

  select, input, button {
    padding: 8px;
    font-size: 16px;
  }

  select[multiple] {
    height: 100px;
  }

  button {
    background-color: #4CAF50;
    color: white;
    border: none;
    cursor: pointer;
    padding: 10px;
  }

  button:hover {
    background-color: #45a049;
  }
</style>