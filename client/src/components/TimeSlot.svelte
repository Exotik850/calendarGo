<script lang="ts">
  export let timeSlot: TimeSlotType;


  function formatDate(dateString: string) {
    const date = new Date(dateString);
    return date.toLocaleString('en-US', {
      hour: 'numeric',
      minute: 'numeric',
      hour12: true
    });
  }

  function formatDateTitle(dateString: string) {
    const date = new Date(dateString);
    return date.toLocaleString('en-US', {
      weekday: 'long',
      month: 'short',
      day: 'numeric',
    });
  }

  function formatDuration(start: string, end: string) {
    const startTime = new Date(start);
    const endTime = new Date(end);
    const durationMs = endTime.getTime() - startTime.getTime();
    const hours = Math.floor(durationMs / (1000 * 60 * 60));
    const minutes = Math.floor((durationMs % (1000 * 60 * 60)) / (1000 * 60));
    return `${hours}h ${minutes}m`;
  }
</script>

<div class="time-slot">
  <div class="date-info">
    <h2>{formatDateTitle(timeSlot.Start)}</h2>
    <p class="duration">{formatDuration(timeSlot.Start, timeSlot.End)}</p>
  </div>
  <div class="time-info">
    <p>Start: {formatDate(timeSlot.Start)}</p>
    <p>End: {formatDate(timeSlot.End)}</p>
  </div>
  {#if timeSlot.ComesAfter.Summary || timeSlot.ComesBefore.Summary}
    <div class="adjacent-events">
      {#if timeSlot.ComesAfter.Summary}
        <div class="event">
          <strong>After:</strong> {timeSlot.ComesAfter.Summary}
          {#if timeSlot.ComesAfter.Location}
            <br>
            <small>{timeSlot.ComesAfter.Location}</small>
          {/if}
        </div>
      {/if}
      {#if timeSlot.ComesBefore.Summary}
        <div class="event">
          <strong>Before:</strong> {timeSlot.ComesBefore.Summary}
          {#if timeSlot.ComesBefore.Location}
            <br>
            <small>{timeSlot.ComesBefore.Location}</small>
          {/if}
        </div>
      {/if}
    </div>
  {/if}
  <div class="distance">
    <p>Distance Added: {(timeSlot.Distance / 1609.0).toLocaleString()} mi</p>
  </div>
</div>

<style>
  .time-slot {
    background-color: #ffffff;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    padding: 16px;
    margin-bottom: 16px;
    font-family: Arial, sans-serif;
  }

  .date-info {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
  }

  h2 {
    margin: 0;
    font-size: 1.5em;
    color: #333;
  }

  .duration {
    font-size: 1.2em;
    font-weight: bold;
    color: #4a90e2;
  }

  .time-info p {
    margin: 4px 0;
    color: #666;
  }

  .adjacent-events {
    margin-top: 12px;
    padding-top: 12px;
    border-top: 1px solid #eee;
  }

  .event {
    margin-bottom: 8px;
  }

  .event strong {
    color: #333;
  }

  .event small {
    color: #888;
  }

  .distance {
    margin-top: 12px;
    font-style: italic;
    color: #666;
  }
</style>