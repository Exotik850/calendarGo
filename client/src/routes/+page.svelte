<script lang="ts">
  import { onMount } from "svelte";
  import TimeSlot from "../components/TimeSlot.svelte";

  interface Calendar {
    [key: string]: string;
  }

  // Summary - ID
  let calendars: Calendar = {};
  // Summaries
  let selectedCalendars: string[] = [];
  let slots: TimeSlotType[] = [];
  let numDays = 7;
  let duration = 60; // minutes
  let eventLoc = "";
  let startLoc = "";
  let text = "";

  function getAuth() {
    return document.cookie
      .split("; ")
      .find((row) => row.startsWith("authCodeEvPlanner"))
      ?.split("=")[1];
  }

  // onMount(async () => {
  //   // Check if the cookie "authCodeEvPlanner" exists
  //   getCalendars();
  // });

  async function getCalendars() {
    if (!getAuth()) {
      // Redirect to the login page
      console.log("No auth code found. Redirecting to login page");
      window.location.href = "/api/login";
      return;
    }
    let ids = await fetch("/listCalendars");
    if (ids.status === 401) {
      console.log("Unauthorized. Redirecting to login page");
      window.location.href = "/api/removecookie";
      return;
    }
    if (ids.status !== 200) {
      console.log("Error fetching calendars:", ids);
      return;
    }
    let idJson = await ids.json();
    for (let [id, value] of Object.entries(idJson)) {
      if (typeof id !== "string" || typeof value !== "string") {
        console.log("Invalid calendar entry:", id, value);
        continue;
      }
      calendars[id] = value;
    }
  }

  async function handleSubmit() {
    if (selectedCalendars.length === 0) {
      alert("Please select at least one calendar");
      return;
    }

    if (!getAuth()) {
      // Redirect to the login page
      console.log("No auth code found. Redirecting to login page");
      window.location.href = "/api/login";
      return;
    }

    const formData = {
      CalIds: selectedCalendars,
      NumDays: numDays,
      Duration: duration,
      EventLoc: eventLoc,
      StartLoc: startLoc,
    };

    // You'll handle the HTTP request to the Go server here
    let result = await fetch("/api/queryAvailableSlots", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(formData),
    });
    if (result.status === 401) {
      console.log("Unauthorized. Redirecting to login page");
      window.location.href = "/api/removecookie";
      return;
    }
    if (result.status !== 200) {
      console.log("Error fetching available spots:", await result.text());
      return;
    }

    slots = await result.json();
  }
</script>

<main>
  <h1>Event Scheduler</h1>

  <form on:submit|preventDefault={handleSubmit}>
    <div>
      {#await getCalendars()}
        <p>Loading calendars...</p>
      {:then}
        <label>
          Select Calendars:
          <select multiple bind:value={selectedCalendars}>
            {#each Object.entries(calendars) as [summary, key] (key)}
              <option value={key}>{summary}</option>
            {/each}
          </select>
        </label>
      {:catch error}
        <p>Error: {error.message}</p>
      {/await}
    </div>

    <div>
      <label>
        Number of Days to Search:
        <input type="number" bind:value={numDays} min="1" max="30" />
      </label>
    </div>

    <div>
      <label>
        Event Duration (minutes):
        <input type="number" bind:value={duration} min="15" step="15" />
      </label>
    </div>

    <div>
      <label>
        Event Location:
        <input type="text" bind:value={eventLoc} placeholder="Enter address" />
      </label>
    </div>

    <div>
      <label>
        Start Location:
        <input type="text" bind:value={startLoc} placeholder="Enter address" />
      </label>
    </div>

    <button type="submit">Find Best Spot</button>
  </form>

  {#each slots as timeSlot}
    <TimeSlot {timeSlot} />
  {/each}
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

  select,
  input,
  button {
    padding: 8px;
    font-size: 16px;
  }

  select[multiple] {
    height: 100px;
  }

  button {
    background-color: #4caf50;
    color: white;
    border: none;
    cursor: pointer;
    padding: 10px;
  }

  button:hover {
    background-color: #45a049;
  }
</style>
