<script lang="ts">
  import { onMount } from "svelte";
  import TimeSlot from "../components/TimeSlot.svelte";
  import { writable } from "svelte/store";
    import toast, { Toaster, type Renderable } from "svelte-french-toast";

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

  const authStore = writable({
    isAuthenticated: false,
    isLoading: true,
  });

  
  async function checkAuthStatus() {
    authStore.set({ isAuthenticated: false, isLoading: true });
    try {
      const response = await fetch("/api/authStatus");
      if (response.ok) {
        authStore.set({ isAuthenticated: true, isLoading: false });
        await getCalendars();
      } else {
        authStore.set({ isAuthenticated: false, isLoading: false });
      }
    } catch (error) {
      errorToast("Error checking auth status: " + error);
      
      authStore.set({ isAuthenticated: false, isLoading: false });
    }
  }

  function errorToast(message: Renderable) {
    console.log("Error: ", message);
    toast.error(message, {
      duration: 4000,
    })
  }

  onMount(async () => {
    // Check if the cookie "authCodeEvPlanner" exists
    await checkAuthStatus();
  });

  async function getCalendars() {
    try {
      const response = await fetch("/api/listCalendars");
      if (response.ok) {
        calendars = await response.json();
      } else if (response.status === 401) {
        authStore.set({ isAuthenticated: false, isLoading: false });
      }
    } catch (error) {
      errorToast("Error fetching calendars: " + error);
    }
  }

  async function handleSubmit() {
    if (selectedCalendars.length === 0) {
      // alert("Please select at least one calendar");
      toast("Please select at least one calendar", {
        duration: 1500,
        icon: "📅",
      });
      return;
    }

    const formData = {
      CalIds: selectedCalendars,
      NumDays: numDays,
      Duration: duration,
      EventLoc: eventLoc,
      StartLoc: startLoc,
    };

    try {
      const result = await fetch("/api/queryAvailableSlots", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(formData),
      });

      if (result.ok) {
        slots = await result.json();
        console.log("Slots: ", slots);
      } else if (result.status === 401) {
        authStore.set({ isAuthenticated: false, isLoading: false });
      } else {
        const text = await result.text();
        errorToast("Error fetching available spots: " + text);
      }
    } catch (error) {
      errorToast("Error submitting form: " + error);
    }
  }

  async function handleSubmitOld() {
    if (selectedCalendars.length === 0) {
      // alert("Please select at least one calendar");
      toast("Please select at least one calendar", {
        duration: 1500,
        icon: "📅",
      });
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

<!-- <main>
  <h1>Event Scheduler</h1>

  <form on:submit|preventDefault={handleSubmit}>
    <div>
      {#if $authStore.isLoading}
        <p>Loading calendars...</p>
      {:else if !$authStore.isAuthenticated}
        <a href="/api/login">Login with Google</a>
      {:else}
        <label>
          Select Calendars:
          <select multiple bind:value={selectedCalendars}>
            {#each Object.entries(calendars) as [summary, key] (key)}
              <option value={key}>{summary}</option>
            {/each}
          </select>
        </label>
      {/if}
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
</style> -->


<main>
  {#if $authStore.isLoading}
    <div class="loading">Loading...</div>
  {:else if !$authStore.isAuthenticated}
    <div class="login-prompt">
      <p>Please log in to use the Event Scheduler.</p>
      <a href="/api/login" class="button">Log In</a>
    </div>
  {:else}
    <h1>Event Scheduler</h1>

    <form on:submit|preventDefault={handleSubmit} class="scheduler-form">
      <div class="form-group">
        <label for="calendars">Select Calendars:</label>
        <select id="calendars" multiple bind:value={selectedCalendars}>
          {#each Object.entries(calendars) as [summary, key] (key)}
            <option value={key}>{summary}</option>
          {/each}
        </select>
      </div>

      <div class="form-group">
        <label for="numDays">Number of Days to Search:</label>
        <input type="range" id="numDays" bind:value={numDays} min="1" max="30" step="1" />
        <span>{numDays} days</span>
      </div>

      <div class="form-group">
        <label for="duration">Event Duration:</label>
        <input type="range" id="duration" bind:value={duration} min="15" max="240" step="15" />
        <span>{duration} minutes</span>
      </div>

      <div class="form-group">
        <label for="eventLoc">Event Location:</label>
        <input type="text" id="eventLoc" bind:value={eventLoc} placeholder="Enter address" />
      </div>

      <div class="form-group">
        <label for="startLoc">Start Location:</label>
        <input type="text" id="startLoc" bind:value={startLoc} placeholder="Enter address" />
      </div>

      <button type="submit" class="button" disabled={$authStore.isLoading}>
        {$authStore.isLoading ? 'Searching...' : 'Find Best Spot'}
      </button>
    </form>

    {#if slots.length > 0}
      <div class="results">
        <h2>Available Time Slots:</h2>
        {#each slots as timeSlot}
          <TimeSlot {timeSlot} />
        {/each}
      </div>
    {:else if !$authStore.isLoading}
      <p class="no-results">No available slots found. Try adjusting your search parameters.</p>
    {/if}
  {/if}
</main>
<Toaster />

<style>
  main {
    max-width: 600px;
    margin: 0 auto;
    padding: 20px;
    font-family: Arial, sans-serif;
  }

  h1 {
    color: #333;
    text-align: center;
  }

  .scheduler-form {
    background: #f9f9f9;
    padding: 20px;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  }

  .form-group {
    margin-bottom: 20px;
  }

  label {
    display: block;
    margin-bottom: 5px;
    color: #555;
  }

  input[type="text"],
  select {
    width: 100%;
    padding: 8px;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 16px;
  }

  input[type="range"] {
    width: 100%;
    margin-top: 5px;
  }

  select[multiple] {
    height: 100px;
  }

  .button {
    background-color: #4caf50;
    color: white;
    border: none;
    padding: 10px 20px;
    text-align: center;
    text-decoration: none;
    display: inline-block;
    font-size: 16px;
    margin: 4px 2px;
    transition-duration: 0.4s;
    cursor: pointer;
    border-radius: 4px;
  }

  .button:hover {
    background-color: #45a049;
  }

  .button:disabled {
    background-color: #cccccc;
    cursor: not-allowed;
  }

  .loading, .login-prompt {
    text-align: center;
    margin-top: 50px;
  }

  .results {
    margin-top: 30px;
  }

  .no-results {
    text-align: center;
    color: #666;
    margin-top: 20px;
  }
</style>