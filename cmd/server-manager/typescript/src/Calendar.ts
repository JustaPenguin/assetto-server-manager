import { Calendar } from '@fullcalendar/core';
import timeGridPlugin from '@fullcalendar/timegrid';
import listPlugin from '@fullcalendar/list';
import bootstrapPlugin from '@fullcalendar/bootstrap';

document.addEventListener('DOMContentLoaded', function() {
    let calendarEl = document.getElementById('calendar');

    if (!calendarEl) {
        return
    }

    let calendar = new Calendar(calendarEl, {
        plugins: [ timeGridPlugin, listPlugin, bootstrapPlugin ],
        defaultView: 'timeGridThreeDay',
        events: '/calendar.json',
        themeSystem: 'bootstrap',

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

        eventRender: function(info) {
            let $title = $(info.el).find('.fc-title');
            let $time = $(info.el).find('.fc-time');


            if (info.event.extendedProps.signUpURL) {
                $time.append('<a class="calendar-signup-link" href="'+info.event.extendedProps.signUpURL+'">Event Sign Up</a>')
            }

            $title.append('<div class="hr-line-solid-no-margin"></div><span class="calendar-small">'+info.event.extendedProps.description+'</span></div>');

            if (info.event.extendedProps.scheduledServerID) {
                $title.append('<div class="calendar-small">On <span class="scheduled-server-id" data-server-id="'+info.event.extendedProps.scheduledServerID+'">another server</span></div>');
            }

            let $listTitle = $(info.el).find('.fc-list-item-title');

            if (info.event.extendedProps.signUpURL) {
                $listTitle.append('</div><a class="calendar-signup-link" href="'+info.event.extendedProps.signUpURL+'">Event Sign Up</a>')
            }

            $listTitle.append('<div class="ml-2"></div><span class="calendar-small">'+info.event.extendedProps.description+'</span></div>');

            if (info.event.extendedProps.scheduledServerID) {
                $listTitle.append('<div class="calendar-small">On <span class="scheduled-server-id" data-server-id="'+info.event.extendedProps.scheduledServerID+'">another server</span></div>');
            }
        },

        nowIndicator: true,
        allDaySlot: false,
        timeGridEventMinHeight: 100,
        height: 800,

        contentHeight: 1000,
    });

    calendar.render();
});
