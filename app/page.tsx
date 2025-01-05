"use client";

import { useEffect, useState } from "react";
import { Card, CardBody } from "@nextui-org/card";
import { Button } from "@nextui-org/button";
import { Progress } from "@nextui-org/progress";
import { Tabs, Tab } from "@nextui-org/tabs";

import { siteConfig } from "@/config/site";

interface Video {
  id: string;
  path: string;
  resolution?: string;
  bitrate?: string;
  status: string;
  originalSize: number;
}

export default function Home() {
  const [unprocessedVideos, setUnprocessedVideos] = useState<Video[]>([]);
  const [currentVideo, setCurrentVideo] = useState<Video | null>(null);
  const [selectedResolution, setSelectedResolution] =
    useState<string>("Original");
  const [selectedBitrate, setSelectedBitrate] = useState<string>("Original");
  const [loading, setLoading] = useState(true);

  const resolutions = [
    { label: "Original", value: "Original" },
    { label: "1080p", value: "1080p" },
    { label: "720p", value: "720p" },
    { label: "480p", value: "480p" },
  ];

  const bitrates = [
    { label: "Original", value: "Original" },
    { label: "h264", value: "h264" },
    { label: "h264 (1000k)", value: "h2641000k" },
  ];

  useEffect(() => {
    fetchUnprocessedVideos();
  }, []);

  const fetchUnprocessedVideos = async () => {
    try {
      setLoading(true);
      const response = await fetch(siteConfig.baseUrl + "/videos/unprocessed");
      const data = await response.json();

      setUnprocessedVideos(data);
      if (data.length > 0 && !currentVideo) {
        setCurrentVideo(data[0]);
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error("Error fetching videos:", error);
    } finally {
      setLoading(false);
    }
  };

  const handleProcessVideo = async () => {
    if (!currentVideo || !selectedResolution || !selectedBitrate) return;

    try {
      await fetch(siteConfig.baseUrl + "/videos/process", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          ...currentVideo,
          resolution: selectedResolution,
          bitrate: selectedBitrate,
          status: "pending",
        }),
      });

      // Remove current video from list and set next one
      const nextVideos = unprocessedVideos.filter(
        (v) => v.id !== currentVideo.id,
      );

      setUnprocessedVideos(nextVideos);
      setCurrentVideo(nextVideos[0] || null);
      setSelectedResolution("Original");
      setSelectedBitrate("Original");
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error("Error processing video:", error);
    }
  };

  const handleDelete = async (video: Video) => {
    try {
      const response = await fetch(siteConfig.baseUrl + "/videos", {
        method: "DELETE",
        headers: {
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ id: video.id }),
      });

      if (!response.ok) {
        throw new Error("Failed to delete video");
      }

      // Обновляем список видео после удаления
      setUnprocessedVideos(unprocessedVideos.filter((v) => v.id !== video.id));
      if (currentVideo?.id === video.id) {
        setCurrentVideo(null);
      }
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error("Error deleting video:", error);
    }
  };

  if (loading) {
    return (
      <div className="flex justify-center items-center h-[calc(100vh-200px)]">
        <Progress
          isIndeterminate
          aria-label="Loading..."
          className="max-w-md"
          size="sm"
        />
      </div>
    );
  }

  if (!currentVideo) {
    return (
      <div className="flex justify-center items-center h-[calc(100vh-200px)]">
        <Card>
          <CardBody>
            <p className="text-center text-gray-400">No videos to process</p>
          </CardBody>
        </Card>
      </div>
    );
  }

  return (
    <div className="flex flex-col lg:flex-row gap-8">
      {/* Video Preview Section */}
      <div className="w-full lg:w-2/3">
        <Card className="w-full h-full">
          <CardBody>
            <div className="aspect-video bg-gray-800 rounded-lg overflow-hidden">
              {/* eslint-disable-next-line jsx-a11y/media-has-caption */}
              <video
                controls
                className="w-full h-full object-contain"
                src={currentVideo.path}
              />
            </div>
            <div className="mt-4">
              <h3 className="text-lg font-semibold text-gray-200">
                {currentVideo.path}
              </h3>
              <p className="text-sm text-gray-400">
                Size: {Math.round(currentVideo.originalSize / 1024 / 1024)}MB
              </p>
            </div>
          </CardBody>
        </Card>
      </div>

      {/* Parameters Section */}
      <div className="w-full lg:w-1/3">
        <Card className="w-full">
          <CardBody className="flex flex-col gap-6">
            <h2 className="text-xl font-semibold text-gray-200">
              Conversion Parameters
            </h2>

            <div className="space-y-4">
              <Tabs
                aria-label="Dynamic tabs"
                color={"primary"}
                items={resolutions}
                selectedKey={selectedResolution}
                onSelectionChange={(e) => {
                  setSelectedResolution(String(e));
                }}
              >
                {(item) => <Tab key={item.value} title={item.label} />}
              </Tabs>

              <br />

              <Tabs
                aria-label="Dynamic tabs"
                color={"primary"}
                items={bitrates}
                selectedKey={selectedBitrate}
                onSelectionChange={(e) => {
                  setSelectedBitrate(String(e));
                }}
              >
                {(item) => <Tab key={item.value} title={item.label} />}
              </Tabs>

              <div className="flex gap-2 mt-4">
                <Button
                  className="w-full"
                  color="primary"
                  isDisabled={!selectedResolution || !selectedBitrate}
                  size="lg"
                  onPress={handleProcessVideo}
                >
                  Process Video
                </Button>
                <Button
                  color="danger"
                  size="lg"
                  onPress={() => handleDelete(currentVideo)}
                >
                  Delete
                </Button>
              </div>

              <div className="mt-4">
                <p className="text-sm text-gray-400">
                  Videos remaining: {unprocessedVideos.length}
                </p>
              </div>
            </div>
          </CardBody>
        </Card>
      </div>
    </div>
  );
}
