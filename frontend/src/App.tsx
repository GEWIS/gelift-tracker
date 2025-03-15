import { MapContainer } from 'react-leaflet/MapContainer'
import { TileLayer } from 'react-leaflet/TileLayer'
import './App.css'
import {useEffect, useState} from "react";
import {Polyline} from "react-leaflet";

interface Datapoint {
    user: string;
    latitude: number;
    longitude: number;
    battery: number;
    time: Date;
}

function App() {
    const [datapoints, setDatapoints] = useState<Datapoint[]>([]);

    const [groupedDatapoints, setGroupedDatapoints] = useState<Record<string, Datapoint[]>>({});

    useEffect(() => {
        fetch("/api/tracks")
            .then(res => res.json())
            .then(data => {
                setDatapoints(data.map((d: any) => {
                    return {
                        user: d.user,
                        latitude: d.latitude,
                        longitude: d.longitude,
                        battery: d.battery,
                        time: new Date(d.timestamp*1000)
                    }
                }))
            })
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
                <Polyline positions={groupedDatapoints[Object.keys(groupedDatapoints)[0]]?.map(d => [d.latitude, d.longitude]) || []} />
            </MapContainer>
        </>
    )
}

export default App
