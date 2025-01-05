"use client";

import { useEffect, useState } from "react";
import { Card, CardBody } from "@nextui-org/card";
import { Progress } from "@nextui-org/progress";
import {
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
} from "@nextui-org/table";
import { Chip } from "@nextui-org/chip";
import { Pagination } from "@nextui-org/pagination";

import { siteConfig } from "@/config/site";

interface Video {
  id: string;
  path: string;
  resolution?: string;
  bitrate?: string;
  status: string;
  originalSize: number;
}

interface Queue {
  pending: Video[];
  processing: Video[];
  completed: Video[];
  total: number;
}

interface PaginationInfo {
  page: number;
  pageSize: number;
  total: number;
  pages: number;
}

export default function QueuePage() {
  const [queue, setQueue] = useState<Queue>({
    pending: [],
    processing: [],
    completed: [],
    total: 0,
  });
  const [loading, setLoading] = useState(true);
  const [currentPage, setCurrentPage] = useState(1);
  const [paginationInfo, setPaginationInfo] = useState<PaginationInfo>({
    page: 1,
    pageSize: 10,
    total: 0,
    pages: 0,
  });
  const [selectedStatus, setSelectedStatus] = useState<string>("all");

  useEffect(() => {
    fetchQueue();
  }, [currentPage, selectedStatus]);

  const fetchQueue = async () => {
    try {
      setLoading(true);
      const status = selectedStatus === "all" ? "" : selectedStatus;
      const response = await fetch(
        siteConfig.baseUrl +
          `/queue?page=${currentPage}&pageSize=10&status=${status}`,
      );
      const data = await response.json();

      setQueue(data.queue);
      setPaginationInfo(data.pagination);
    } catch (error) {
      // eslint-disable-next-line no-console
      console.error("Error fetching queue:", error);
    } finally {
      setLoading(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case "pending":
        return "warning";
      case "processing":
        return "primary";
      case "completed":
        return "success";
      default:
        return "default";
    }
  };

  const statusCounts = {
    all: queue.total,
    pending: queue.pending.length,
    processing: queue.processing.length,
    completed: queue.completed.length,
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

  const allVideos = [...queue.pending, ...queue.processing, ...queue.completed];

  if (allVideos.length === 0) {
    return (
      <div className="flex justify-center items-center h-[calc(100vh-200px)]">
        <Card>
          <CardBody>
            <p className="text-center text-gray-400">No videos in queue</p>
          </CardBody>
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-8">
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        {Object.entries({
          All: statusCounts.all,
          Pending: statusCounts.pending,
          Processing: statusCounts.processing,
          Completed: statusCounts.completed,
        }).map(([status, count]) => (
          <Card
            key={status}
            isPressable
            className={
              selectedStatus === status.toLowerCase()
                ? "border-2 border-primary"
                : ""
            }
            onPress={() => {
              setSelectedStatus(status.toLowerCase());
              setCurrentPage(1);
            }}
          >
            <CardBody>
              <h3 className="text-lg font-semibold text-gray-200 mb-2">
                {status}
              </h3>
              <p
                className={`text-3xl font-bold text-${getStatusColor(status.toLowerCase())}`}
              >
                {count}
              </p>
            </CardBody>
          </Card>
        ))}
      </div>

      <Card>
        <CardBody>
          <div className="flex flex-col gap-4">
            <Table aria-label="Video queue">
              <TableHeader>
                <TableColumn>File</TableColumn>
                <TableColumn>Size</TableColumn>
                <TableColumn>Resolution</TableColumn>
                <TableColumn>Bitrate</TableColumn>
                <TableColumn>Status</TableColumn>
              </TableHeader>
              <TableBody>
                {allVideos.map((video) => (
                  <TableRow key={video.id}>
                    <TableCell>
                      <span className="font-medium text-gray-200">
                        {video.path}
                      </span>
                    </TableCell>
                    <TableCell>
                      {Math.round(video.originalSize / 1024 / 1024)}MB
                    </TableCell>
                    <TableCell>{video.resolution || "-"}</TableCell>
                    <TableCell>{video.bitrate || "-"}</TableCell>
                    <TableCell>
                      <Chip
                        color={getStatusColor(video.status)}
                        size="sm"
                        variant="flat"
                      >
                        {video.status}
                      </Chip>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>

            <div className="flex justify-center mt-4">
              <Pagination
                showControls
                color="primary"
                page={currentPage}
                total={paginationInfo.pages}
                variant="bordered"
                onChange={setCurrentPage}
              />
            </div>
          </div>
        </CardBody>
      </Card>
    </div>
  );
}
