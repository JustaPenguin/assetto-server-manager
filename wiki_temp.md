_Championships let you track points for drivers/classes/teams across multiple events! We built championships from the ground up for Assetto Corsa, here I'll go through some of the basics of using them._

***

When you install Server Manager for the first time it comes with some example championships already set up, feel free to have a look and a mess with them to figure the system out. You'll probably find championships easiest to learn by experimenting with them yourself, but if you'd rather follow a guide then here it is:

# Create a new Championship

### Open/Closed

Championships can take two main forms, open or closed. After the championship name this is the first choice you need to make in the new championship form.

*Closed championships* (default) require a complete entry list in order to start, this entry list can be created manually or built from a sign up form. If you already know who is taking part in your championship the best method is to close the championship and then manually add each entrants GUID to the entry list. If you want to allow people to sign up to the championship then it's best to turn on the sign up form, which I'll cover in more detail later. 

*Open championships* will start without a complete entry list (a list of cars is still required, but the GUID field can be empty), and will add drivers to the championship automatically if they join the server whilst the championship is running. This can let you run championships that anybody can join, but can result in odd championship results, after all there's a good reason you don't see new drivers in F1 every week.

*Open championships* also have the option to persist entrants. This means that when a driver joins the server whilst the championship is running they will be permanently added to the entry list, and will lock the slot that they are added to, essentially becoming a locked entrant in the championship.

### Override Password

Pretty self-explanatory, you can override whatever the server password is normally set to for all championship events. You can use this to set exclusive passwords for certain championships, or turn the password off for open championships.

### Important Information

You can use this markdown input for anything you want - links to cars/tracks/required mods, information about race formats/rules etc. Alongside Content Manager integration this information will be placed in the server description in Content Manager whilst the championship (or a practice session for the championship) is running.

### Sign Up Form

Sign up Forms let you manage sign ups for a championship, you can ask users for their email and team, let them choose a car and skin, make them complete a reCAPTCHA in order to sign up and add any extra questions to ask them as part of the process. You can then decide whether all applications should be approved by admin/write users or be automatically accepted. 

When a user goes to the sign up link they are prompted to enter their name, GUID and complete any other of the enabled optional fields. A user can also sign in with Steam at this point to fill some of the information automatically. If the user has an account on your manager then their details can be automatically entered from there too!

Sign Up Forms can be a great way of gauging community interest in a series, and assures drivers that they have a spot in your championship!

### Entrants, Classes, Points

Entrants are separated into multiple "Class" cards. The championship will track points per class, allowing you to easily run championships with multiple car classes, for example you could run a race with 4 LMP1 cars, 6 LMP2 cars and 10 GTE cars with individual points tracking per class. Each class can have multiple cars and entrants, and points can be managed per class.

Cars added to the car list will be available to each entrant, the "any available car" option will _randomly_ select a car for each entrant. Each entrant can be configured to have a certain skin, ballast, restrictor or fixed setup. You can also set the entrant's team here, bearing in mind that points are also tracked per team as well as per driver (as in F1). The GUID of an entrant refers to their Steam profile GUID, also commonly referred to as the SteamID64. (Once you've added an entrant to any event in your manager they will be added to the autofill list at the top of the entrant card to make adding them easier for future events).

Points can be configured for each finishing position in a race, you can also add modifiers for other events such as Best Lap, Pole Position, Collisions and more!

### Adding an Event

Once you've configured the championship itself it's time to add some events. You can add two different types of events to a championship, a normal event or a race weekend. You can always add/remove events later on, but you need at least one event to finish creating the championship.

_Normal events_ are equivalent to Custom Races, which I won't go into detail on here, the only difference being that your entry list is taken straight from the championship itself.

_Race Weekends_ allow you to create more complicated event flows using filters and sorting options on the entry list between sessions, for example you could properly recreate the format of a full F1 weekend. If you look at the Race Weekends page of your manager you will see a few examples. Again race weekends added to championships are created in exactly the same way as normal Race Weekends, so I won't cover that here. It is important to note in Race Weekends you can configure points for every individual session whilst you are creating them, although race events will have the championship events imported by default.

# Working with Existing Championships

### Overview

Once a championship has been created you can view the championship page, this will contain a table of current championship standings, entrants, a points reference, controls to edit the championship or add more events (you can import existing Custom Races and Race Weekends as well as create new ones) and event cards for each event.

### Event Cards

Each event card can be sorted on the page simply by clicking and dragging, you can also change the sorting and hide/show completed/uncompleted events using the controls above the event cards.

**If an event has not yet been started** the event card will contain controls for starting or scheduling the event, starting a practice session for the event and managing the event. It will also show which sessions the event has and how long they are, if the event is scheduled the card will show the scheduled time.

_Scheduling_ allows you to set a date for the event to start, this should work fine in your timezone. You can also use recurrence rules to make an event run at regular intervals. In a championship recurring events will make a copy of themselves in the event list each time they recur.

_Practice Sessions_ are looping practice sessions that will keep running until either stopped/another event is manually started or the championship event is started. Whilst these practice sessions are running the lap times across all sessions will be collected and displayed in the Live Timings. The idea here is to get people trying to set competitive laps in the run-up to a championship event.

_The Manage Event_ button gives you a few options. You can edit the event setup, delete the event or import results files to the event. Importing results is sometimes necessary if the championship event missed the end of a session, or if something else may have gone wrong. So long as you have the results files you can import them to the championship easily here! 

**If an event is in progress** the event card will show which session is currently in progress and allows you to restart/cancel the event.

**If an event has been completed** the event card will show a small overview of the results (top three positions, fastest laps in qualifying and the race). It also has a button to show more detailed results information in a table (much like normal results pages, which are also linked to for each session), and a manage event button that allows you to import results or delete the event.

***

_And I think that's everything! I hope you found this introductory guide to championships useful, and that you enjoy using them to compete with fellow sim racers!_