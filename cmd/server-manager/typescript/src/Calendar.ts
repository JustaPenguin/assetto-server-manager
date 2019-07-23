import { Calendar } from '@fullcalendar/core';
import timeGridPlugin from '@fullcalendar/timegrid';
import listPlugin from '@fullcalendar/list';

document.addEventListener('DOMContentLoaded', function() {
    let calendarEl = document.getElementById('calendar');

    if (!calendarEl) {
        return
    }

    let calendar = new Calendar(calendarEl, {
        plugins: [ timeGridPlugin, listPlugin ],
        defaultView: 'timeGridThreeDay',
        events: '/calendar.json',

        header: {
            center: 'timeGridWeek,timeGridThreeDay,listWeek' // buttons for switching between views
        },
        views: {
            timeGridThreeDay: {
                type: 'timeGrid',
                duration: { days: 3 },
                buttonText: '3 day'
            }
        },

        nowIndicator: true,
        allDaySlot: false,
        timeGridEventMinHeight: 100, // @TODO scroll on overflow?
        aspectRatio: 1,
    });

    calendar.render();
});