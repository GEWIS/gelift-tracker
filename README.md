## GELIFT tracker
GELIFT weekend is a hitchhiking competition where teams compete against each other to be the fastest at the location. 
Challenges can be completed to receive time deduction. 
GELIFT tracker is the piece of software that is used publish the current location of the teams in the GELIFT weekend 
in real time to both the participants and the rest of the world to see who is the closest to the finish. 

### System design
#### System requirements
The app is build on three main goals in order of importance:
1. **Availability, accuracy and precision of the locations.** The location is used for safety as well, so this needs
to be available, accurate and precise at all times.
2. **Ease of use for the participants.** The set-up should be as minimal as possible to make sure they use this app
instead of the other common solutions such as Life360 or WhatsApp.
3. **Mobile first frontend.** The frontend of the application should be easily accesible on mobile and on poor WiFi, as
it should be possible to reach while on the road (for the parcipants) and the link is generally shared of messages apps
on mobile.

#### System architecture
The app works on three main components, as visualized below.
![System architecutre](./assets/System%20architecture.drawio.png)

- **Location publisher**: For the location publisher the [OwnTracks](https://owntracks.org/) app is used,
this app can function in two modes, namely MQTT and HTTP mode. For the first requirement, MQTT has been chosen, as it 
off-loads some of this availability to existing applications. What is known for OwnTracks as "user" have been defined 
as the teams, and what OwnTracks calls "devices" have been called "participants". So for each participant "Pluk" from
the team "Pettelet" they publish to `/owntracks/petteflet/pluk` with the messages described in the OwnTracks documentation.
- **MQTT Broker**: As suggested by OwnTracks, the [Mosquitto broker](https://mosquitto.org/) is used. Its responsibility 
is making sure that the location from the publisher arrive at the server, and are published to the Go backend. MQTT has 
some techniques for this as described in the protocol specification. 
- **Location API**: The Location API is a piece of custom code that makes sure that the locations as received from 
the MQTT Broker are stored in an format that can be used by the frontend. This includes things like the team,
the participant, the battery percentage, the time, but most importantly the location. The location API does not calculate any extra
  properties, but does make sure that every entry is unique in terms of MQTT message. Finally, it also exposes a 
single HTTP endpoint for serving the tracks that are stored in the database. The endpoint is public and it is 
possible to create multiple frontends for displaying the tracks. 
[GORM](https://gorm.io/), [Echo](https://echo.labstack.com/) and [Paho](https://github.com/eclipse-paho/paho.golang) 
are used as libraries for the webserver, ORM and MQTT client respectively.
- **Location visualizer**: The location visualizer is the React app that displays the tracks. It takes the tracks
from the location API, groups this data by the user/team, it sorts the data on timestamp, 
calculates properties like distance to the destination. It also contains some settings for customizing visualization.
For the maps [Leaflet](https://react-leaflet.js.org/) + 
[OpenStreetMaps](https://www.openstreetmap.org/) is used, 
[primereact](https://primereact.org/) + [tailwindcss](https://tailwindcss.com/) for components and styling.

### Changelogs
#### v2025
- Initial release
- Visualize current location and history
- Switch between individual view and team view
- Switch permanent tooltips
- Switch on/off specific teams
- Info on who published and when
- Distance till destination and arrival time

### Future ideas
- Clean up the code
- Backoffice to create/update/delete teams and assign start times to the teams to be used by the GELIFT committee.
- Timeline to scroll back in time