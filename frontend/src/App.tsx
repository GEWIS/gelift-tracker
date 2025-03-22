import { MapContainer } from 'react-leaflet/MapContainer'
import { TileLayer } from 'react-leaflet/TileLayer'
import './App.css'
import {useEffect, useState} from "react";
import {CircleMarker, Polyline, Popup, Tooltip} from "react-leaflet";
import uniqolor from "uniqolor";

interface Datapoint {
    user: string;
    teamName: string;
    latitude: number;
    longitude: number;
    battery: number;
    time: Date;
}

function App() {
    const [datapoints, setDatapoints] = useState<Datapoint[]>([]);

    const [groupedDatapoints, setGroupedDatapoints] = useState<Record<string, Datapoint[]>>({});

    function getTracks() {
        fetch("/api/tracks")
            .then(res => res.json())
            .then(data => {
                setDatapoints(data.map((d: any) => {
                    return {
                        user: d.user,
                        teamName: d.user,
                        latitude: d.latitude,
                        longitude: d.longitude,
                        battery: d.battery,
                        time: new Date(d.timestamp*1000)
                    }
                }))
            })
    }

    useEffect(() => {
        getTracks();

        const interval = setInterval(getTracks, 10*1000);

        return () => clearInterval(interval);
    }, [])

    useEffect(() => {
        const uniqueTeams = new Set<string>();

        for (const point of datapoints) {
            uniqueTeams.add(point.user);
        }

        const _groupedDatapoints: Record<string, Datapoint[]> = {};

        for (const uniqueTeam of uniqueTeams) {
            _groupedDatapoints[uniqueTeam] = datapoints.filter(d => d.user === uniqueTeam);
        }

        console.log(_groupedDatapoints);
        setGroupedDatapoints(_groupedDatapoints);
    }, [datapoints])

    return (
        <>
            <MapContainer className={"map-container"} center={[52.278,7.973]} zoom={8}>
                <TileLayer
                    attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                    url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                />
                {
                    Object.values(groupedDatapoints).map((p) => {
                        const lastPos = p[p.length-1]
                        const color = uniqolor(lastPos.user, { lightness: 45 }).color
                        return <>
                            <CircleMarker center={[lastPos.latitude, lastPos.longitude]} pathOptions={{ color }} radius={10}>
                                <Popup>
                                    <b>Last position details:</b> <br/>
                                    Battery: { lastPos.battery } <br/>
                                    Time: { lastPos.time.toLocaleString() }
                                </Popup>
                                <Tooltip direction="bottom" offset={[0, 10]} opacity={1} permanent>
                                    { lastPos.user }
                                </Tooltip>
                            </CircleMarker>
                            <Polyline
                                pathOptions={{ color }}
                                positions={p.map(d => [d.latitude, d.longitude])} />
                        </>
                    })
                }
            </MapContainer>
        </>
    )
}

export default App
