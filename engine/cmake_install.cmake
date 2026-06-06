# Install script for directory: /Users/niko/dev/go/aivisioninference/engine

# Set the install prefix
if(NOT DEFINED CMAKE_INSTALL_PREFIX)
  set(CMAKE_INSTALL_PREFIX "/usr/local")
endif()
string(REGEX REPLACE "/$" "" CMAKE_INSTALL_PREFIX "${CMAKE_INSTALL_PREFIX}")

# Set the install configuration name.
if(NOT DEFINED CMAKE_INSTALL_CONFIG_NAME)
  if(BUILD_TYPE)
    string(REGEX REPLACE "^[^A-Za-z0-9_]+" ""
           CMAKE_INSTALL_CONFIG_NAME "${BUILD_TYPE}")
  else()
    set(CMAKE_INSTALL_CONFIG_NAME "")
  endif()
  message(STATUS "Install configuration: \"${CMAKE_INSTALL_CONFIG_NAME}\"")
endif()

# Set the component getting installed.
if(NOT CMAKE_INSTALL_COMPONENT)
  if(COMPONENT)
    message(STATUS "Install component: \"${COMPONENT}\"")
    set(CMAKE_INSTALL_COMPONENT "${COMPONENT}")
  else()
    set(CMAKE_INSTALL_COMPONENT)
  endif()
endif()

# Is this installation the result of a crosscompile?
if(NOT DEFINED CMAKE_CROSSCOMPILING)
  set(CMAKE_CROSSCOMPILING "FALSE")
endif()

# Set path to fallback-tool for dependency-resolution.
if(NOT DEFINED CMAKE_OBJDUMP)
  set(CMAKE_OBJDUMP "/usr/bin/objdump")
endif()

if(CMAKE_INSTALL_COMPONENT STREQUAL "Unspecified" OR NOT CMAKE_INSTALL_COMPONENT)
  file(INSTALL DESTINATION "${CMAKE_INSTALL_PREFIX}/bin" TYPE EXECUTABLE FILES "/Users/niko/dev/go/aivisioninference/engine/aivision-engine")
  if(EXISTS "$ENV{DESTDIR}${CMAKE_INSTALL_PREFIX}/bin/aivision-engine" AND
     NOT IS_SYMLINK "$ENV{DESTDIR}${CMAKE_INSTALL_PREFIX}/bin/aivision-engine")
    if(CMAKE_INSTALL_DO_STRIP)
      execute_process(COMMAND "/usr/bin/strip" -u -r "$ENV{DESTDIR}${CMAKE_INSTALL_PREFIX}/bin/aivision-engine")
    endif()
  endif()
endif()

if(CMAKE_INSTALL_COMPONENT STREQUAL "Unspecified" OR NOT CMAKE_INSTALL_COMPONENT)
  include("/Users/niko/dev/go/aivisioninference/engine/CMakeFiles/aivision-engine.dir/install-cxx-module-bmi-noconfig.cmake" OPTIONAL)
endif()

if(CMAKE_INSTALL_COMPONENT STREQUAL "Unspecified" OR NOT CMAKE_INSTALL_COMPONENT)
  file(INSTALL DESTINATION "${CMAKE_INSTALL_PREFIX}/include/aivision-engine" TYPE FILE FILES
    "/Users/niko/dev/go/aivisioninference/engine/include/engine.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/ipc/transport.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/ipc/ipc_server.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/ipc/heartbeat.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/pipeline/hw_buffer.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/pipeline/ring_queue.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/pipeline/snapshot.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/pipeline/worker_pool.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/pipeline/hal.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/pipeline/pipeline.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/pipeline/pipeline_manager.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/pipeline/encoder_stage.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/pipeline/inference_stage.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/pipeline/rtsp_push_stage.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/algo/abi_contract.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/algo/so_handle.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/algo/algo_instance.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/algo/algo_manager.h"
    "/Users/niko/dev/go/aivisioninference/engine/include/monitor/metrics_reporter.h"
    )
endif()

string(REPLACE ";" "\n" CMAKE_INSTALL_MANIFEST_CONTENT
       "${CMAKE_INSTALL_MANIFEST_FILES}")
if(CMAKE_INSTALL_LOCAL_ONLY)
  file(WRITE "/Users/niko/dev/go/aivisioninference/engine/install_local_manifest.txt"
     "${CMAKE_INSTALL_MANIFEST_CONTENT}")
endif()
if(CMAKE_INSTALL_COMPONENT)
  if(CMAKE_INSTALL_COMPONENT MATCHES "^[a-zA-Z0-9_.+-]+$")
    set(CMAKE_INSTALL_MANIFEST "install_manifest_${CMAKE_INSTALL_COMPONENT}.txt")
  else()
    string(MD5 CMAKE_INST_COMP_HASH "${CMAKE_INSTALL_COMPONENT}")
    set(CMAKE_INSTALL_MANIFEST "install_manifest_${CMAKE_INST_COMP_HASH}.txt")
    unset(CMAKE_INST_COMP_HASH)
  endif()
else()
  set(CMAKE_INSTALL_MANIFEST "install_manifest.txt")
endif()

if(NOT CMAKE_INSTALL_LOCAL_ONLY)
  file(WRITE "/Users/niko/dev/go/aivisioninference/engine/${CMAKE_INSTALL_MANIFEST}"
     "${CMAKE_INSTALL_MANIFEST_CONTENT}")
endif()
