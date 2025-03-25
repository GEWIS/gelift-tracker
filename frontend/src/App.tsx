import { MapContainer } from 'react-leaflet/MapContainer'
import { TileLayer } from 'react-leaflet/TileLayer'
import './App.css'
import {useEffect, useState} from "react";
import {CircleMarker, Polyline, Popup, Tooltip} from "react-leaflet";
import uniqolor from "uniqolor";
import { Panel } from 'primereact/panel';
import {InputSwitch} from "primereact/inputswitch";
import distance from "@turf/distance"

interface Datapoint {
    team: string;
    user: string;
    teamName: string;
    latitude: number;
    longitude: number;
    battery: number;
    time: Date;
    distanceLeft: number;
}

function getColor(name: string) {
    return uniqolor(name, { lightness: [35, 50] }).color
}

function App() {
    const [datapoints, setDatapoints] = useState<Datapoint[]>([]);

    const [groupedDatapoints, setGroupedDatapoints] = useState<Record<string, Datapoint[]>>({});

    const [groupByTeam, setGroupByTeam] = useState<boolean>(true);

    function getTracks() {
        fetch("/api/tracks")
            .then(res => res.json())
            .then(data => {
                setDatapoints(data.map((d: any) => {
                    return {
                        team: d.team,
                        user: d.user,
                        teamName: d.user,
                        latitude: d.latitude,
                        longitude: d.longitude,
                        battery: d.battery,
                        time: new Date(d.timestamp*1000),
                        distanceLeft: distance([d.longitude, d.latitude] /*, [5.4762087, 51.4435241] */ , [10.003503, 53.5655278] )
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
        const uniqueEntities = new Set<string>();

        for (const point of datapoints) {
            if (groupByTeam) {
                uniqueEntities.add(point.team);
            } else {
                uniqueEntities.add(point.user);
            }
        }

        const _groupedDatapoints: Record<string, Datapoint[]> = {};

        for (const uniqueEntity of uniqueEntities) {
            _groupedDatapoints[uniqueEntity] = datapoints.filter(d => {
                return groupByTeam && d.team === uniqueEntity
                    || d.user === uniqueEntity
            });
        }

        setGroupedDatapoints(_groupedDatapoints);
    }, [datapoints, groupByTeam])

    return (
        <>
            { /* Overlay */ }
            <div className={"absolute flex flex-row justify-end w-full"}>
                <div className={"flex flex-col w-2xs"}>
                    <Panel header={"Controls"} className={"bg-white flex flex-col m-1 z-500 bg rounded"} toggleable collapsed={true}>
                        <div className={"flex flex-row"}>
                            <InputSwitch
                                checked={!groupByTeam}
                                onChange={(e) => setGroupByTeam(!e.value)}
                            />
                            <div className={"ml-2"}>Individual view</div>
                        </div>
                    </Panel>
                    <Panel header={"Stats"} className={"bg-white flex flex-col mx-1 z-500 bg rounded"} toggleable collapsed={true}>
                        <div className={"grid grid-cols-2 justify-items-start"}>
                            { Object.values(groupedDatapoints)
                                .sort((a, b) => {
                                    return a[a.length-1].distanceLeft - b[b.length-1].distanceLeft
                                })
                                .map((p) => {
                                    p.sort((a, b) => a.time.getTime()-b.time.getTime())
                                    const lastPos = p[p.length-1]
                                    const color = getColor(groupByTeam ? lastPos.team : lastPos.user)
                                    return <>
                                        <div style={{ color }}>
                                            { groupByTeam ? lastPos.team : lastPos.user }
                                        </div>
                                        <div>
                                            { Math.round(lastPos.distanceLeft) } km left
                                        </div>
                                    </>
                                })
                            }
                        </div>
                    </Panel>
                </div>
            </div>
            <MapContainer className={"map-container z-10"} center={[52.278,7.973]} zoom={8}>
                <TileLayer
                    attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a> contributors'
                    url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
                />
                {
                    Object.values(groupedDatapoints).map((p) => {
                        p.sort((a, b) => a.time.getTime()-b.time.getTime())
                        const lastPos = p[p.length-1]
                        const color = getColor(groupByTeam ? lastPos.team : lastPos.user)
                        console.log(lastPos);
                        return <>
                            <CircleMarker center={[lastPos.latitude, lastPos.longitude]} pathOptions={{ color }} radius={10}>
                                <Popup>
                                    <b>Last position details:</b> <br/>
                                    Time: { lastPos.time.toLocaleString() } <br/>
                                    Reported by: { lastPos.user } <br/>
                                    Battery: { lastPos.battery } <br/>
                                </Popup>
                                <Tooltip direction="bottom" offset={[0, 10]} opacity={1} permanent>
                                    { groupByTeam ? lastPos.team : lastPos.user }
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
