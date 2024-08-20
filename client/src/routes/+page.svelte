<script lang="ts">
  import { onMount } from 'svelte';

  // Summary - ID 
  let calendars: Map<string, string> = new Map();
  // Summaries
  let selectedCalendars: string[] = [];
  let numDays = 7;
  let duration = 60; // minutes
  let eventLocation = '';
  let startLocation = '';

  onMount(async () => {
    // Fetch available calendars from the server
    // This is a placeholder - you'll need to implement the actual API call
    calendars = new Map([
      ["1", 'Calendar 1'],
      ["2", 'Calendar 2'],
      ["3", 'Calendar 3']
    ]);
  });

  function handleSubmit() {
    const formData = {
      calendars: selectedCalendars,
      numDays,
      duration,
      eventLocation,
      startLocation
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
        <input type="text" bind:value={eventLocation} placeholder="Enter address">
      </label>
    </div>

    <div>
      <label>
        Start Location:
        <input type="text" bind:value={startLocation} placeholder="Enter address">
      </label>
    </div>

    <button type="submit">Find Best Spot</button>
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