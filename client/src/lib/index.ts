// place files you want to import through the `$lib` alias in this folder.

interface TimeSlotType {
  Year: number;
  Month: number;
  Day: number;
  Start: string;
  End: string;
  ComesAfter: {
    Summary: string;
    Location: string;
  };
  ComesBefore: {
    Summary: string;
    Location: string;
  };
  Distance: number;
}