/*++

Copyright (c) Microsoft Corporation

Abstract:

    This module contains a sample implementation of an indirect display driver. See the included README.md file and the
    various TODO blocks throughout this file and all accompanying files for information on building a production driver.

    MSDN documentation on indirect displays can be found at https://msdn.microsoft.com/en-us/library/windows/hardware/mt761968(v=vs.85).aspx.

Environment:

    User Mode, UMDF

    For tracing log, use Inflight Trace Record (IFR) https://docs.microsoft.com/en-us/windows-hardware/drivers/wdf/using-wpp-software-tracing-in-kmdf-and-umdf-2-drivers
    // https://docs.microsoft.com/en-us/windows-hardware/drivers/devtest/adding-wpp-software-tracing-to-a-windows-driver#step-5-instrument-the-driver-code-to-generate-trace-messages-at-appropriate-points
--*/

#include <tchar.h>

#include "Driver.h"
#include "Driver.tmh"
#include "Public.h"

//
// Define an Interface Guid for QuickDesk Virtual Display device class.
// This GUID is used to register (IoRegisterDeviceInterface)
// an instance of an interface so that user application
// can control the QuickDesk Virtual Display device.
//
const GUID GUID_DEVINTERFACE_IDD_DRIVER_DEVICE = \
    { 0x781EF630, 0x72B2, 0x11d2, { 0xB8, 0x52,  0x00,  0xC0,  0x4E,  0xAF,  0x52,  0x72 } };
//{781EF630-72B2-11d2-B852-00C04EAF5272}


using namespace std;
using namespace Microsoft::IndirectDisp;
using namespace Microsoft::WRL;

#pragma region SampleMonitors

static constexpr DWORD IDD_SAMPLE_MONITOR_COUNT = 3; // If monitor count > ARRAYSIZE(s_SampleMonitors), we create edid-less monitors

// Default modes reported for edid-less monitors. The first mode is set as preferred
static const struct IndirectSampleMonitor::SampleMonitorMode s_SampleDefaultModes[] = 
{
    { 1920, 1080, 60 },
    { 1600,  900, 60 },
    { 1024,  768, 75 },
};

// FOR SAMPLE PURPOSES ONLY, Static info about monitors that will be reported to OS
static const struct IndirectSampleMonitor s_SampleMonitors[] =
{
    // Modified EDID from Dell S2719DGF
    {
        {
            0x00,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0x00,0x10,0xAC,0xE6,0xD0,0x55,0x5A,0x4A,0x30,0x24,0x1D,0x01,
            0x04,0xA5,0x3C,0x22,0x78,0xFB,0x6C,0xE5,0xA5,0x55,0x50,0xA0,0x23,0x0B,0x50,0x54,0x00,0x02,0x00,
            0xD1,0xC0,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x58,0xE3,0x00,
            0xA0,0xA0,0xA0,0x29,0x50,0x30,0x20,0x35,0x00,0x55,0x50,0x21,0x00,0x00,0x1A,0x00,0x00,0x00,0xFF,
            0x00,0x37,0x4A,0x51,0x58,0x42,0x59,0x32,0x0A,0x20,0x20,0x20,0x20,0x20,0x00,0x00,0x00,0xFC,0x00,
            0x53,0x32,0x37,0x31,0x39,0x44,0x47,0x46,0x0A,0x20,0x20,0x20,0x20,0x00,0x00,0x00,0xFD,0x00,0x28,
            0x9B,0xFA,0xFA,0x40,0x01,0x0A,0x20,0x20,0x20,0x20,0x20,0x20,0x00,0x2C
        },
        {
            { 2560, 1440, 144 },
            { 1920, 1080,  60 },
            { 1024,  768,  60 },
        },
        0
    },
    // Modified EDID from Lenovo Y27fA
    {
        {
            0x00,0xFF,0xFF,0xFF,0xFF,0xFF,0xFF,0x00,0x30,0xAE,0xBF,0x65,0x01,0x01,0x01,0x01,0x20,0x1A,0x01,
            0x04,0xA5,0x3C,0x22,0x78,0x3B,0xEE,0xD1,0xA5,0x55,0x48,0x9B,0x26,0x12,0x50,0x54,0x00,0x08,0x00,
            0xA9,0xC0,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x01,0x68,0xD8,0x00,
            0x18,0xF1,0x70,0x2D,0x80,0x58,0x2C,0x45,0x00,0x53,0x50,0x21,0x00,0x00,0x1E,0x00,0x00,0x00,0x10,
            0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0x00,0xFD,0x00,
            0x30,0x92,0xB4,0xB4,0x22,0x01,0x0A,0x20,0x20,0x20,0x20,0x20,0x20,0x00,0x00,0x00,0xFC,0x00,0x4C,
            0x45,0x4E,0x20,0x59,0x32,0x37,0x66,0x41,0x0A,0x20,0x20,0x20,0x00,0x11
        },
        {
            { 3840, 2160,  60 },
            { 1600,  900,  60 },
            { 1024,  768,  60 },
        },
        0
    }
    // Another EDID
    // https://github.com/roshkins/IddSampleDriver/blob/df7238c1f242e1093cdcab0ea749f34094570283/IddSampleDriver/Driver.cpp#L419
};

#pragma endregion

#pragma region helpers

static inline void FillSignalInfo(DISPLAYCONFIG_VIDEO_SIGNAL_INFO& Mode, DWORD Width, DWORD Height, DWORD VSync, bool bMonitorMode)
{
    Mode.totalSize.cx = Mode.activeSize.cx = Width;
    Mode.totalSize.cy = Mode.activeSize.cy = Height;

    // See https://docs.microsoft.com/en-us/windows/win32/api/wingdi/ns-wingdi-displayconfig_video_signal_info
    Mode.AdditionalSignalInfo.vSyncFreqDivider = bMonitorMode ? 0 : 1;
    Mode.AdditionalSignalInfo.videoStandard = 255;

    Mode.vSyncFreq.Numerator = VSync;
    Mode.vSyncFreq.Denominator = 1;
    Mode.hSyncFreq.Numerator = VSync * Height;
    Mode.hSyncFreq.Denominator = 1;

    Mode.scanLineOrdering = DISPLAYCONFIG_SCANLINE_ORDERING_PROGRESSIVE;

    Mode.pixelRate = ((UINT64) VSync) * ((UINT64) Width) * ((UINT64) Height);
}

static IDDCX_MONITOR_MODE CreateIddCxMonitorMode(DWORD Width, DWORD Height, DWORD VSync, IDDCX_MONITOR_MODE_ORIGIN Origin = IDDCX_MONITOR_MODE_ORIGIN_DRIVER)
{
    IDDCX_MONITOR_MODE Mode = {};

    Mode.Size = sizeof(Mode);
    Mode.Origin = Origin;
    FillSignalInfo(Mode.MonitorVideoSignalInfo, Width, Height, VSync, true);

    return Mode;
}

static IDDCX_TARGET_MODE CreateIddCxTargetMode(DWORD Width, DWORD Height, DWORD VSync)
{
    IDDCX_TARGET_MODE Mode = {};

    Mode.Size = sizeof(Mode);
    FillSignalInfo(Mode.TargetVideoSignalInfo.targetVideoSignalInfo, Width, Height, VSync, false);

    return Mode;
}

#pragma endregion

extern "C" DRIVER_INITIALIZE DriverEntry;

EVT_WDF_DRIVER_UNLOAD RustDeskIddDriverUnload;
EVT_WDF_DRIVER_DEVICE_ADD IddRustDeskDeviceAdd;
EVT_WDF_DEVICE_D0_ENTRY IddRustDeskDeviceD0Entry;

// https://docs.microsoft.com/en-us/windows-hardware/drivers/kernel/defining-i-o-control-codes
EVT_IDD_CX_DEVICE_IO_CONTROL IddRustDeskIoDeviceControl;

EVT_IDD_CX_ADAPTER_INIT_FINISHED IddRustDeskAdapterInitFinished;
EVT_IDD_CX_ADAPTER_COMMIT_MODES IddRustDeskAdapterCommitModes;

EVT_IDD_CX_PARSE_MONITOR_DESCRIPTION IddRustDeskParseMonitorDescription;
EVT_IDD_CX_MONITOR_GET_DEFAULT_DESCRIPTION_MODES IddRustDeskMonitorGetDefaultModes;
EVT_IDD_CX_MONITOR_QUERY_TARGET_MODES IddRustDeskMonitorQueryModes;

EVT_IDD_CX_MONITOR_ASSIGN_SWAPCHAIN IddRustDeskMonitorAssignSwapChain;
EVT_IDD_CX_MONITOR_UNASSIGN_SWAPCHAIN IddRustDeskMonitorUnassignSwapChain;

struct IndirectDeviceContextWrapper
{
    IndirectDeviceContext* pContext;

    void Cleanup()
    {
        delete pContext;
        pContext = nullptr;
    }
};

struct IndirectMonitorContextWrapper
{
    IndirectMonitorContext* pContext;

    void Cleanup()
    {
        delete pContext;
        pContext = nullptr;
    }
};

// This macro creates the methods for accessing an IndirectDeviceContextWrapper as a context for a WDF object
WDF_DECLARE_CONTEXT_TYPE(IndirectDeviceContextWrapper);

WDF_DECLARE_CONTEXT_TYPE(IndirectMonitorContextWrapper);

extern "C" BOOL WINAPI DllMain(
    _In_ HINSTANCE hInstance,
    _In_ UINT dwReason,
    _In_opt_ LPVOID lpReserved)
{
    UNREFERENCED_PARAMETER(hInstance);
    UNREFERENCED_PARAMETER(lpReserved);
    UNREFERENCED_PARAMETER(dwReason);

    return TRUE;
}

_Use_decl_annotations_
extern "C" NTSTATUS DriverEntry(
    PDRIVER_OBJECT  pDriverObject,
    PUNICODE_STRING pRegistryPath
)
{
    WDF_DRIVER_CONFIG Config;
    NTSTATUS Status;

    WDF_OBJECT_ATTRIBUTES Attributes;
    WDF_OBJECT_ATTRIBUTES_INIT(&Attributes);

    WPP_INIT_TRACING(pDriverObject, pRegistryPath);

    WDF_DRIVER_CONFIG_INIT(&Config,
        IddRustDeskDeviceAdd
    );
    Config.EvtDriverUnload = RustDeskIddDriverUnload;

    // The WdfDriverCreate method creates a framework driver object for the calling driver.
    Status = WdfDriverCreate(pDriverObject, pRegistryPath, &Attributes, &Config, WDF_NO_HANDLE);
    if (!NT_SUCCESS(Status))
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DRIVER,
            "%!FUNC! cannot create device %!STATUS!",
            Status);
        WPP_CLEANUP(WdfDriverWdmGetDriverObject(Driver));
    }
    else
    {
        TraceEvents(TRACE_LEVEL_INFORMATION,
            TRACE_DRIVER,
            "%!FUNC! driver created with path %wZ",
            pRegistryPath);
    }

    return Status;
}

_Use_decl_annotations_
void RustDeskIddDriverUnload(_In_ WDFDRIVER /*Driver*/)
{
    WPP_CLEANUP(WdfDriverWdmGetDriverObject(Driver));
}

// https://community.osr.com/discussion/290895/how-to-realize-hot-plug-function-of-virtual-display-with-indirect-display
// Hi,
// in Indirect display driver, after use WdfDeviceCreateDeviceInterface to create guid, 
// you should use IddCxDeviceInitialize, IddDeviceIoControl is not the same as other wdf's DeviceIoControl function,
// his first argument is WDFDEVICE Device,not WDFQUEUE,
// so you don't need to use WdfIoQueueCreate to create a queue to receive I / O queue message.
// After all of the above, you should create IOCTL code, this can establish communication between applicationand idd device.
_Use_decl_annotations_
NTSTATUS IddRustDeskDeviceAdd(WDFDRIVER Driver, PWDFDEVICE_INIT pDeviceInit)
{
    NTSTATUS Status = STATUS_SUCCESS;
    WDF_PNPPOWER_EVENT_CALLBACKS PnpPowerCallbacks;

    UNREFERENCED_PARAMETER(Driver);

    // Register for power callbacks - in this sample only power-on is needed
    WDF_PNPPOWER_EVENT_CALLBACKS_INIT(&PnpPowerCallbacks);
    PnpPowerCallbacks.EvtDeviceD0Entry = IddRustDeskDeviceD0Entry;
    WdfDeviceInitSetPnpPowerEventCallbacks(pDeviceInit, &PnpPowerCallbacks);

    IDD_CX_CLIENT_CONFIG IddConfig;
    IDD_CX_CLIENT_CONFIG_INIT(&IddConfig);

    // If the driver wishes to handle custom IoDeviceControl requests, it's necessary to use this callback since IddCx
    // redirects IoDeviceControl requests to an internal queue. This sample does not need this.
    // https://docs.microsoft.com/en-us/windows-hardware/drivers/display/iddcx-objects
    IddConfig.EvtIddCxDeviceIoControl = IddRustDeskIoDeviceControl;

    IddConfig.EvtIddCxAdapterInitFinished = IddRustDeskAdapterInitFinished;

    IddConfig.EvtIddCxParseMonitorDescription = IddRustDeskParseMonitorDescription;
    IddConfig.EvtIddCxMonitorGetDefaultDescriptionModes = IddRustDeskMonitorGetDefaultModes;
    IddConfig.EvtIddCxMonitorQueryTargetModes = IddRustDeskMonitorQueryModes;
    IddConfig.EvtIddCxAdapterCommitModes = IddRustDeskAdapterCommitModes;
    IddConfig.EvtIddCxMonitorAssignSwapChain = IddRustDeskMonitorAssignSwapChain;
    IddConfig.EvtIddCxMonitorUnassignSwapChain = IddRustDeskMonitorUnassignSwapChain;

    Status = IddCxDeviceInitConfig(pDeviceInit, &IddConfig);
    if (!NT_SUCCESS(Status))
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DRIVER,
            "%!FUNC! cannot init device config %!STATUS!",
            Status);
        return Status;
    }

    WDF_OBJECT_ATTRIBUTES Attr;
    WDF_OBJECT_ATTRIBUTES_INIT_CONTEXT_TYPE(&Attr, IndirectDeviceContextWrapper);
    Attr.EvtCleanupCallback = [](WDFOBJECT Object)
    {
        // Automatically cleanup the context when the WDF object is about to be deleted
        auto* pContext = WdfObjectGet_IndirectDeviceContextWrapper(Object);
        if (pContext)
        {
            pContext->Cleanup();
        }
    };

    WDFDEVICE Device = nullptr;
    Status = WdfDeviceCreate(&pDeviceInit, &Attr, &Device);
    if (!NT_SUCCESS(Status))
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! cannot create device %!STATUS!",
            Status);
        return Status;
    }

    //
    // Create device interface for this device. The interface will be
    // enabled by the framework when we return from StartDevice successfully.
    // Clients of this driver will open this interface and send ioctls.
    //
    Status = WdfDeviceCreateDeviceInterface(
        Device,
        &GUID_DEVINTERFACE_IDD_DRIVER_DEVICE,
        NULL // No Reference String. If you provide one it will appended to the
    );   // symbolic link. Some drivers register multiple interfaces for the same device
         // and use the reference string to distinguish between them
    if (!NT_SUCCESS(Status))
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! WdfDeviceCreateDeviceInterface failed %!STATUS!",
            Status);
        return Status;
    }

    Status = IddCxDeviceInitialize(Device);
    if (!NT_SUCCESS(Status))
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! cannot initialize device %!STATUS!",
            Status);
        return Status;
    }

    // Create a new device context object and attach it to the WDF device object
    auto* pContext = WdfObjectGet_IndirectDeviceContextWrapper(Device);
    pContext->pContext = new IndirectDeviceContext(Device);

    return Status;
}

_Use_decl_annotations_
NTSTATUS IddRustDeskDeviceD0Entry(WDFDEVICE Device, WDF_POWER_DEVICE_STATE PreviousState)
{
    UNREFERENCED_PARAMETER(PreviousState);

    // This function is called by WDF to start the device in the fully-on power state.

    auto* pContext = WdfObjectGet_IndirectDeviceContextWrapper(Device);
    pContext->pContext->InitAdapter();

    return STATUS_SUCCESS;
}

#pragma region Direct3DDevice

Direct3DDevice::Direct3DDevice(LUID AdapterLuid) : AdapterLuid(AdapterLuid)
{

}

Direct3DDevice::Direct3DDevice()
{
    AdapterLuid = LUID{};
}

HRESULT Direct3DDevice::Init()
{
    // The DXGI factory could be cached, but if a new render adapter appears on the system, a new factory needs to be
    // created. If caching is desired, check DxgiFactory->IsCurrent() each time and recreate the factory if !IsCurrent.
    HRESULT hr = CreateDXGIFactory2(0, IID_PPV_ARGS(&DxgiFactory));
    if (FAILED(hr))
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DRIVER,
            "%!FUNC! cannot create dxgi factory2 %!HRESULT!",
            hr);
        return hr;
    }

    // Find the specified render adapter
    hr = DxgiFactory->EnumAdapterByLuid(AdapterLuid, IID_PPV_ARGS(&Adapter));
    if (FAILED(hr))
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DRIVER,
            "%!FUNC! cannot enum adapter by luid %!HRESULT!",
            hr);
        return hr;
    }

    // Create a D3D device using the render adapter. BGRA support is required by the WHQL test suite.
    hr = D3D11CreateDevice(
        Adapter.Get(),
        D3D_DRIVER_TYPE_UNKNOWN,
        nullptr,
        D3D11_CREATE_DEVICE_BGRA_SUPPORT,
        nullptr,
        0,
        D3D11_SDK_VERSION,
        &Device,
        nullptr,
        &DeviceContext);
    if (FAILED(hr))
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DRIVER,
            "%!FUNC! cannot create d3d11 device %!HRESULT!",
            hr);

        // If creating the D3D device failed, it's possible the render GPU was lost (e.g. detachable GPU) or else the
        // system is in a transient state.
        return hr;
    }

    return S_OK;
}

#pragma endregion

#pragma region SwapChainProcessor

SwapChainProcessor::SwapChainProcessor(IDDCX_SWAPCHAIN hSwapChain, shared_ptr<Direct3DDevice> Device, HANDLE NewFrameEvent)
    : m_hSwapChain(hSwapChain), m_Device(Device), m_hAvailableBufferEvent(NewFrameEvent)
{
    m_hTerminateEvent.Attach(CreateEvent(nullptr, FALSE, FALSE, nullptr));

    // Immediately create and run the swap-chain processing thread, passing 'this' as the thread parameter
    m_hThread.Attach(CreateThread(nullptr, 0, RunThread, this, 0, nullptr));
}

SwapChainProcessor::~SwapChainProcessor()
{
    // Alert the swap-chain processing thread to terminate
    SetEvent(m_hTerminateEvent.Get());

    if (m_hThread.Get())
    {
        // Wait for the thread to terminate
        WaitForSingleObject(m_hThread.Get(), INFINITE);
    }
}

DWORD CALLBACK SwapChainProcessor::RunThread(LPVOID Argument)
{
    reinterpret_cast<SwapChainProcessor*>(Argument)->Run();
    return 0;
}

void SwapChainProcessor::Run()
{
    // For improved performance, make use of the Multimedia Class Scheduler Service, which will intelligently
    // prioritize this thread for improved throughput in high CPU-load scenarios.
    DWORD AvTask = 0;
    HANDLE AvTaskHandle = AvSetMmThreadCharacteristics(_T("Distribution"), &AvTask);

    RunCore();

    // Always delete the swap-chain object when swap-chain processing loop terminates in order to kick the system to
    // provide a new swap-chain if necessary.
    WdfObjectDelete((WDFOBJECT)m_hSwapChain);
    m_hSwapChain = nullptr;

    AvRevertMmThreadCharacteristics(AvTaskHandle);
}

void SwapChainProcessor::RunCore()
{
    // Get the DXGI device interface
    ComPtr<IDXGIDevice> DxgiDevice;
    HRESULT hr = m_Device->Device.As(&DxgiDevice);
    if (FAILED(hr))
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DRIVER,
            "%!FUNC! cannot convert dxgi device %!HRESULT!",
            hr);
        return;
    }

    IDARG_IN_SWAPCHAINSETDEVICE SetDevice = {};
    SetDevice.pDevice = DxgiDevice.Get();

    hr = IddCxSwapChainSetDevice(m_hSwapChain, &SetDevice);
    if (FAILED(hr))
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DRIVER,
            "%!FUNC! failed, swap chain set device %!HRESULT!",
            hr);
        return;
    }

    TraceEvents(TRACE_LEVEL_RESERVED7, TRACE_DRIVER, "%!FUNC! begin acquire and release buffers in a loop");

    // Acquire and release buffers in a loop
    for (;;)
    {
        ComPtr<IDXGIResource> AcquiredBuffer;

        // Ask for the next buffer from the producer
        IDARG_OUT_RELEASEANDACQUIREBUFFER Buffer = {};
        hr = IddCxSwapChainReleaseAndAcquireBuffer(m_hSwapChain, &Buffer);

        // AcquireBuffer immediately returns STATUS_PENDING if no buffer is yet available
        if (hr == E_PENDING)
        {
            // We must wait for a new buffer
            HANDLE WaitHandles [] =
            {
                m_hAvailableBufferEvent,
                m_hTerminateEvent.Get()
            };
            DWORD WaitResult = WaitForMultipleObjects(ARRAYSIZE(WaitHandles), WaitHandles, FALSE, 16);
            if (WaitResult == WAIT_OBJECT_0 || WaitResult == WAIT_TIMEOUT)
            {
                // We have a new buffer, so try the AcquireBuffer again
                continue;
            }
            else if (WaitResult == WAIT_OBJECT_0 + 1)
            {
                // We need to terminate
                TraceEvents(TRACE_LEVEL_RESERVED7,
                    TRACE_DRIVER,
                    "%!FUNC! Terminate");
                break;
            }
            else
            {
                // The wait was cancelled or something unexpected happened
                hr = HRESULT_FROM_WIN32(WaitResult);

                TraceEvents(TRACE_LEVEL_RESERVED6,
                    TRACE_DRIVER,
                    "%!FUNC! The wait was cancelled or something unexpected happened %!HRESULT!",
                    hr);
                break;
            }
        }
        else if (SUCCEEDED(hr))
        {
            // We have new frame to process, the surface has a reference on it that the driver has to release
            AcquiredBuffer.Attach(Buffer.MetaData.pSurface);

            // ==============================
            // TODO: Process the frame here
            //
            // This is the most performance-critical section of code in an IddCx driver. It's important that whatever
            // is done with the acquired surface be finished as quickly as possible. This operation could be:
            //  * a GPU copy to another buffer surface for later processing (such as a staging surface for mapping to CPU memory)
            //  * a GPU encode operation
            //  * a GPU VPBlt to another surface
            //  * a GPU custom compute shader encode operation
            // ==============================

            // We have finished processing this frame hence we release the reference on it.
            // If the driver forgets to release the reference to the surface, it will be leaked which results in the
            // surfaces being left around after swapchain is destroyed.
            // NOTE: Although in this sample we release reference to the surface here; the driver still
            // owns the Buffer.MetaData.pSurface surface until IddCxSwapChainReleaseAndAcquireBuffer returns
            // S_OK and gives us a new frame, a driver may want to use the surface in future to re-encode the desktop 
            // for better quality if there is no new frame for a while
            AcquiredBuffer.Reset();
            
            // Indicate to OS that we have finished inital processing of the frame, it is a hint that
            // OS could start preparing another frame
            hr = IddCxSwapChainFinishedProcessingFrame(m_hSwapChain);
            if (FAILED(hr))
            {
                TraceEvents(TRACE_LEVEL_ERROR,
                    TRACE_DRIVER,
                    "%!FUNC! cannot finish processing frame %!HRESULT!",
                    hr);
                break;
            }

            // ==============================
            // TODO: Report frame statistics once the asynchronous encode/send work is completed
            //
            // Drivers should report information about sub-frame timings, like encode time, send time, etc.
            // ==============================
            // IddCxSwapChainReportFrameStatistics(m_hSwapChain, ...);
        }
        else
        {
            // The swap-chain was likely abandoned (e.g. DXGI_ERROR_ACCESS_LOST), so exit the processing loop
            TraceEvents(TRACE_LEVEL_ERROR,
                TRACE_DRIVER,
                "%!FUNC! The swap-chain was likely abandoned %!HRESULT!",
                hr);
            break;
        }
    }
}

#pragma endregion

#pragma region IndirectDeviceContext

IndirectDeviceContext::IndirectDeviceContext(_In_ WDFDEVICE WdfDevice)
    : m_AdapterInitStatus(STATUS_ERROR_ADAPTER_NOT_INIT)
    , m_WdfDevice(WdfDevice)
{
    m_Adapter = {};

    for (UINT i = 0; i < m_sMaxMonitorCount; i++)
    {
        m_Monitors[i] = NULL;
    }
}

IndirectDeviceContext::~IndirectDeviceContext()
{
    for (UINT i = 0; i < m_sMaxMonitorCount; i++)
    {
        if (m_Monitors[i] != NULL)
        {
            // TODO: Plug out monitor
            m_Monitors[i] = NULL;
        }
    }
}

void IndirectDeviceContext::InitAdapter()
{
    // ==============================
    // TODO: Update the below diagnostic information in accordance with the target hardware. The strings and version
    // numbers are used for telemetry and may be displayed to the user in some situations.
    //
    // This is also where static per-adapter capabilities are determined.
    // ==============================

    IDDCX_ADAPTER_CAPS AdapterCaps = {};
    AdapterCaps.Size = sizeof(AdapterCaps);

    // Declare basic feature support for the adapter (required)
    AdapterCaps.MaxMonitorsSupported = Microsoft::IndirectDisp::IndirectDeviceContext::GetMaxMonitorCount();
    AdapterCaps.EndPointDiagnostics.Size = sizeof(AdapterCaps.EndPointDiagnostics);
    AdapterCaps.EndPointDiagnostics.GammaSupport = IDDCX_FEATURE_IMPLEMENTATION_NONE;
    AdapterCaps.EndPointDiagnostics.TransmissionType = IDDCX_TRANSMISSION_TYPE_WIRED_OTHER;

    // Declare your device strings for telemetry (required)
    AdapterCaps.EndPointDiagnostics.pEndPointFriendlyName = L"QuickDesk Virtual Display";
    AdapterCaps.EndPointDiagnostics.pEndPointManufacturerName = L"QuickCoder";
    AdapterCaps.EndPointDiagnostics.pEndPointModelName = L"QuickDesk Virtual Display";

    // Declare your hardware and firmware versions (required)
    IDDCX_ENDPOINT_VERSION Version = {};
    Version.Size = sizeof(Version);
    Version.MajorVer = 1;
    AdapterCaps.EndPointDiagnostics.pFirmwareVersion = &Version;
    AdapterCaps.EndPointDiagnostics.pHardwareVersion = &Version;

    // Initialize a WDF context that can store a pointer to the device context object
    WDF_OBJECT_ATTRIBUTES Attr;
    WDF_OBJECT_ATTRIBUTES_INIT_CONTEXT_TYPE(&Attr, IndirectDeviceContextWrapper);

    IDARG_IN_ADAPTER_INIT AdapterInit = {};
    AdapterInit.WdfDevice = m_WdfDevice;
    AdapterInit.pCaps = &AdapterCaps;
    AdapterInit.ObjectAttributes = &Attr;

    // Start the initialization of the adapter, which will trigger the AdapterFinishInit callback later
    IDARG_OUT_ADAPTER_INIT AdapterInitOut;
    NTSTATUS Status = IddCxAdapterInitAsync(&AdapterInit, &AdapterInitOut);

    if (NT_SUCCESS(Status))
    {
        // Store a reference to the WDF adapter handle
        m_Adapter = AdapterInitOut.AdapterObject;

        // Store the device context object into the WDF object context
        auto* pContext = WdfObjectGet_IndirectDeviceContextWrapper(AdapterInitOut.AdapterObject);
        pContext->pContext = this;

        TraceEvents(TRACE_LEVEL_INFORMATION,
            TRACE_DEVICE,
            "%!FUNC! init adapter done");
    }
    else
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! cannot init adapter %!STATUS!",
            Status);
    }
}

void IndirectDeviceContext::FinishInit(UINT ConnectorIndex)
{
    WDF_OBJECT_ATTRIBUTES Attr;
    WDF_OBJECT_ATTRIBUTES_INIT_CONTEXT_TYPE(&Attr, IndirectMonitorContextWrapper);

    // In the sample driver, we report a monitor right away but a real driver would do this when a monitor connection event occurs
    IDDCX_MONITOR_INFO MonitorInfo = {};
    MonitorInfo.Size = sizeof(MonitorInfo);
    MonitorInfo.MonitorType = DISPLAYCONFIG_OUTPUT_TECHNOLOGY_HDMI;
    MonitorInfo.ConnectorIndex = ConnectorIndex;

    MonitorInfo.MonitorDescription.Size = sizeof(MonitorInfo.MonitorDescription);
    MonitorInfo.MonitorDescription.Type = IDDCX_MONITOR_DESCRIPTION_TYPE_EDID;
    if (ConnectorIndex >= m_sMaxMonitorCount)
    {
        MonitorInfo.MonitorDescription.DataSize = 0;
        MonitorInfo.MonitorDescription.pData = nullptr;
    }
    else
    {
        MonitorInfo.MonitorDescription.DataSize = IndirectSampleMonitor::szEdidBlock;
        MonitorInfo.MonitorDescription.pData = const_cast<BYTE*>(s_SampleMonitors[0].pEdidBlock);
    }

    // ==============================
    // TODO: The monitor's container ID should be distinct from "this" device's container ID if the monitor is not
    // permanently attached to the display adapter device object. The container ID is typically made unique for each
    // monitor and can be used to associate the monitor with other devices, like audio or input devices. In this
    // sample we generate a random container ID GUID, but it's best practice to choose a stable container ID for a
    // unique monitor or to use "this" device's container ID for a permanent/integrated monitor.
    // ==============================

    // Create a container ID
    CoCreateGuid(&MonitorInfo.MonitorContainerId);

    IDARG_IN_MONITORCREATE MonitorCreate = {};
    MonitorCreate.ObjectAttributes = &Attr;
    MonitorCreate.pMonitorInfo = &MonitorInfo;

    // Create a monitor object with the specified monitor descriptor
    IDARG_OUT_MONITORCREATE MonitorCreateOut;
    NTSTATUS Status = IddCxMonitorCreate(m_Adapter, &MonitorCreate, &MonitorCreateOut);
    if (NT_SUCCESS(Status))
    {
        TraceEvents(TRACE_LEVEL_INFORMATION,
            TRACE_DEVICE,
            "%!FUNC! create monitor done");

        // Create a new monitor context object and attach it to the Idd monitor object
        auto* pMonitorContextWrapper = WdfObjectGet_IndirectMonitorContextWrapper(MonitorCreateOut.MonitorObject);
        pMonitorContextWrapper->pContext = new IndirectMonitorContext(MonitorCreateOut.MonitorObject);

        // Tell the OS that the monitor has been plugged in
        IDARG_OUT_MONITORARRIVAL ArrivalOut;
        Status = IddCxMonitorArrival(MonitorCreateOut.MonitorObject, &ArrivalOut);
        if (NT_SUCCESS(Status))
        {
            TraceEvents(TRACE_LEVEL_INFORMATION,
                TRACE_DEVICE,
                "%!FUNC! tell the OS that the monitor has been plugged in done");
        }
        else
        {
            TraceEvents(TRACE_LEVEL_ERROR,
                TRACE_DEVICE,
                "%!FUNC! cannot tell the OS that the monitor has been plugged in %!STATUS!",
                Status);
        }
    }
    else
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! cannot create monitor %!STATUS!",
            Status);
    }
}

NTSTATUS IndirectDeviceContext::CheckIndex(UINT index)
{
    if (index >= m_sMaxMonitorCount)
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! monitor index %u out of range %u", index, m_sMaxMonitorCount);
        return STATUS_ERROR_INDEX_OOR;
    }
    return STATUS_SUCCESS;
}

#define EARLY_RETURN_NTSTATUS(c) { NTSTATUS ret = (c); if (!NT_SUCCESS(ret)) { return ret; } }

NTSTATUS IndirectDeviceContext::PlugInMonitor(PCtlPlugIn Param)
{
    UINT ConnectorIndex = Param->ConnectorIndex;
    UINT MonitorEDID = Param->MonitorEDID;
    GUID ContainerID = Param->ContainerId;

    EARLY_RETURN_NTSTATUS(CheckIndex(ConnectorIndex));

    TraceEvents(TRACE_LEVEL_INFORMATION,
        TRACE_DEVICE,
        "%!FUNC! begin monitor %u",
        ConnectorIndex);

    if (!NT_SUCCESS(m_AdapterInitStatus))
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! adapter init failed %!STATUS!", m_AdapterInitStatus);
        return m_AdapterInitStatus;
    }

    if (m_Monitors[ConnectorIndex] != NULL)
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! monitor of index %u already exists", ConnectorIndex);
        return STATUS_ERROR_MONITOR_EXISTS;
    }

    WDF_OBJECT_ATTRIBUTES Attr;
    WDF_OBJECT_ATTRIBUTES_INIT_CONTEXT_TYPE(&Attr, IndirectMonitorContextWrapper);

    // In the sample driver, we report a monitor right away but a real driver would do this when a monitor connection event occurs
    IDDCX_MONITOR_INFO MonitorInfo = {};
    MonitorInfo.Size = sizeof(MonitorInfo);
    MonitorInfo.MonitorType = DISPLAYCONFIG_OUTPUT_TECHNOLOGY_HDMI;
    MonitorInfo.ConnectorIndex = ConnectorIndex;

    MonitorInfo.MonitorDescription.Size = sizeof(MonitorInfo.MonitorDescription);
    MonitorInfo.MonitorDescription.Type = IDDCX_MONITOR_DESCRIPTION_TYPE_EDID;
    if (MonitorEDID >= ARRAYSIZE(s_SampleMonitors)) // create edid-less monitor
    {
        MonitorInfo.MonitorDescription.DataSize = 0;
        MonitorInfo.MonitorDescription.pData = nullptr;
    }
    else
    {
        MonitorInfo.MonitorDescription.DataSize = IndirectSampleMonitor::szEdidBlock;
        MonitorInfo.MonitorDescription.pData = const_cast<BYTE*>(s_SampleMonitors[MonitorEDID].pEdidBlock);
    }

    MonitorInfo.MonitorContainerId = ContainerID;

    IDARG_IN_MONITORCREATE MonitorCreate = {};
    MonitorCreate.ObjectAttributes = &Attr;
    MonitorCreate.pMonitorInfo = &MonitorInfo;

    // Create a monitor object with the specified monitor descriptor
    IDARG_OUT_MONITORCREATE MonitorCreateOut;
    NTSTATUS Status = IddCxMonitorCreate(m_Adapter, &MonitorCreate, &MonitorCreateOut);
    if (NT_SUCCESS(Status))
    {
        TraceEvents(TRACE_LEVEL_INFORMATION,
            TRACE_DEVICE,
            "%!FUNC! create monitor done");

        // Tell the OS that the monitor has been plugged in
        IDARG_OUT_MONITORARRIVAL ArrivalOut;
        Status = IddCxMonitorArrival(MonitorCreateOut.MonitorObject, &ArrivalOut);
        if (NT_SUCCESS(Status))
        {
            TraceEvents(TRACE_LEVEL_INFORMATION,
                TRACE_DEVICE,
                "%!FUNC! monitor %u done",
                ConnectorIndex);

            // Create a new monitor context object and attach it to the Idd monitor object
            auto* pMonitorContextWrapper = WdfObjectGet_IndirectMonitorContextWrapper(MonitorCreateOut.MonitorObject);
            pMonitorContextWrapper->pContext = new IndirectMonitorContext(MonitorCreateOut.MonitorObject);
            m_Monitors[ConnectorIndex] = MonitorCreateOut.MonitorObject;
        }
        else
        {
            TraceEvents(TRACE_LEVEL_ERROR,
                TRACE_DEVICE,
                "%!FUNC! cannot tell the OS that the monitor has been plugged in %!STATUS!",
                Status);
            (VOID)IddCxMonitorDeparture(MonitorCreateOut.MonitorObject);
        }
    }
    else
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! cannot create monitor %!STATUS!",
            Status);
    }
    return Status;
}

NTSTATUS IndirectDeviceContext::PlugOutMonitor(PCtlPlugOut Param)
{
    UINT ConnectorIndex = Param->ConnectorIndex;

    EARLY_RETURN_NTSTATUS(CheckIndex(ConnectorIndex));

    TraceEvents(TRACE_LEVEL_INFORMATION,
        TRACE_DEVICE,
        "%!FUNC! begin monitor %u",
        ConnectorIndex);

    if (ConnectorIndex >= m_sMaxMonitorCount)
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! monitor index %u out of range %u", ConnectorIndex, m_sMaxMonitorCount);
        return STATUS_ERROR_INDEX_OOR;
    }

    if (m_Monitors[ConnectorIndex] == NULL)
    {
        return STATUS_ERROR_MONITOR_NOT_EXISTS;
    }
    NTSTATUS Status = IddCxMonitorDeparture(m_Monitors[ConnectorIndex]);
    if (NT_SUCCESS(Status))
    {
        TraceEvents(TRACE_LEVEL_INFORMATION,
            TRACE_DEVICE,
            "%!FUNC! monitor %u done",
            ConnectorIndex);
        m_Monitors[ConnectorIndex] = NULL;
    }
    else
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! failed monitor %u %!STATUS!",
            ConnectorIndex,
            Status);
    }
    return Status;
}

NTSTATUS IndirectDeviceContext::UpdateMonitorModes(PCtlMonitorModes Param)
{
    // Unimplemented
    UINT ConnectorIndex = Param->ConnectorIndex;
    UINT ModeCount = Param->ModeCount;

    EARLY_RETURN_NTSTATUS(CheckIndex(ConnectorIndex));

    TraceEvents(TRACE_LEVEL_INFORMATION,
        TRACE_DEVICE,
        "%!FUNC! begin update monitor mode %u",
        ConnectorIndex);

    if (ConnectorIndex >= m_sMaxMonitorCount)
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! monitor index %u out of range %u", ConnectorIndex, m_sMaxMonitorCount);
        return STATUS_ERROR_INDEX_OOR;
    }

    if (m_Monitors[ConnectorIndex] == NULL)
    {
        return STATUS_ERROR_MONITOR_NOT_EXISTS;
    }

    if (ModeCount == 0)
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! monitor %u ModeCount is 0",
            ConnectorIndex);
        return STATUS_ERROR_MONITOR_INVALID_PARAM;
    }

    IDDCX_TARGET_MODE* PTargetMode = (IDDCX_TARGET_MODE*)malloc(sizeof(IDDCX_TARGET_MODE) * ModeCount);
    if (PTargetMode == NULL)
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! monitor %u alloc failed",
            ConnectorIndex);
        return STATUS_ERROR_MONITOR_OOM;
    }
    for (UINT i = 0; i < ModeCount; ++i)
    {
        PTargetMode[i] = CreateIddCxTargetMode(
            Param->Modes[i].Width,
            Param->Modes[i].Height,
            Param->Modes[i].Sync);
    }
    IDARG_IN_UPDATEMODES UpdateModes{ IDDCX_UPDATE_REASON_OTHER, ModeCount, PTargetMode };
    NTSTATUS Status = IddCxMonitorUpdateModes(m_Monitors[ConnectorIndex], &UpdateModes);
    free(PTargetMode);
    if (NT_SUCCESS(Status))
    {
        TraceEvents(TRACE_LEVEL_INFORMATION,
            TRACE_DEVICE,
            "%!FUNC! monitor %u done",
            ConnectorIndex);
    }
    else
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! monitor %u %!STATUS!",
            ConnectorIndex,
            Status);
    }
    return Status;
}

IndirectMonitorContext::IndirectMonitorContext(_In_ IDDCX_MONITOR Monitor) :
    m_Monitor(Monitor)
{
}

IndirectMonitorContext::~IndirectMonitorContext()
{
    m_ProcessingThread.reset();
}

void IndirectMonitorContext::AssignSwapChain(IDDCX_SWAPCHAIN SwapChain, LUID RenderAdapter, HANDLE NewFrameEvent)
{
    m_ProcessingThread.reset();

    auto Device = make_shared<Direct3DDevice>(RenderAdapter);
    if (FAILED(Device->Init()))
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! cannot init Direct3DDevice");

        // It's important to delete the swap-chain if D3D initialization fails, so that the OS knows to generate a new
        // swap-chain and try again.
        WdfObjectDelete(SwapChain);
    }
    else
    {
        TraceEvents(TRACE_LEVEL_RESERVED7,
            TRACE_DEVICE,
            "%!FUNC! Init Direct3DDevice done");

        // Create a new swap-chain processing thread
        m_ProcessingThread.reset(new SwapChainProcessor(SwapChain, Device, NewFrameEvent));
    }
}

void IndirectMonitorContext::UnassignSwapChain()
{
    // Stop processing the last swap-chain
    m_ProcessingThread.reset();
}

#pragma endregion

#pragma region DDI Callbacks

_Use_decl_annotations_
VOID
IddRustDeskIoDeviceControl(WDFDEVICE Device, WDFREQUEST Request, size_t OutputBufferLength, size_t InputBufferLength, ULONG IoControlCode)
{
    TraceEvents(TRACE_LEVEL_INFORMATION,
        TRACE_DEVICE,
        "%!FUNC! receive io control code %ul, in buf len %u\n",
        IoControlCode,
        (unsigned)InputBufferLength);

    UNREFERENCED_PARAMETER(Request);
    UNREFERENCED_PARAMETER(OutputBufferLength);
    // UNREFERENCED_PARAMETER(InputBufferLength);
    // https://docs.microsoft.com/en-us/windows-hardware/drivers/display/iddcx-objects

    NTSTATUS Status = STATUS_SUCCESS;
    PVOID  Buffer;
    size_t BufSize;
    auto* pContext = WdfObjectGet_IndirectDeviceContextWrapper(Device);

    switch (IoControlCode)
    {
    case IOCTL_QUICKDESK_VD_PLUG_IN:
        PQuickDeskPlugInParams pPlugIn;
        Status = WdfRequestRetrieveInputBuffer(Request, sizeof(QuickDeskPlugInParams), &Buffer, &BufSize);
        if (!NT_SUCCESS(Status)) {
            TraceEvents(TRACE_LEVEL_ERROR,
                TRACE_DEVICE,
                "%!FUNC! cannot retrieve input buffer %!STATUS!",
                Status);
            break;
        }
        {
            // Adapt QuickDesk params to internal CtlPlugIn
            pPlugIn = (PQuickDeskPlugInParams)Buffer;
            CtlPlugIn plugIn;
            plugIn.ConnectorIndex = pPlugIn->ConnectorIndex;
            plugIn.MonitorEDID = pPlugIn->MonitorEDID;
            plugIn.ContainerId = pPlugIn->ContainerId;
            Status = pContext->pContext->PlugInMonitor(&plugIn);
        }
        break;
    case IOCTL_QUICKDESK_VD_PLUG_OUT:
        PQuickDeskPlugOutParams pPlugOut;
        Status = WdfRequestRetrieveInputBuffer(Request, sizeof(QuickDeskPlugOutParams), &Buffer, &BufSize);
        if (!NT_SUCCESS(Status))
        {
            TraceEvents(TRACE_LEVEL_ERROR,
                TRACE_DEVICE,
                "%!FUNC! cannot retrieve input buffer %!STATUS!",
                Status);
            break;
        }
        {
            pPlugOut = (PQuickDeskPlugOutParams)Buffer;
            CtlPlugOut plugOut;
            plugOut.ConnectorIndex = pPlugOut->ConnectorIndex;
            Status = pContext->pContext->PlugOutMonitor(&plugOut);
        }
        break;
    case IOCTL_QUICKDESK_VD_UPDATE_MODE:
        PQuickDeskUpdateModeParams pUpdateMode;
        Status = WdfRequestRetrieveInputBuffer(Request, InputBufferLength, &Buffer, &BufSize);
        if (!NT_SUCCESS(Status))
        {
            TraceEvents(TRACE_LEVEL_ERROR,
                TRACE_DEVICE,
                "%!FUNC! cannot retrieve input buffer %!STATUS!",
                Status);
            break;
        }
        {
            pUpdateMode = (PQuickDeskUpdateModeParams)Buffer;
            // Reinterpret as CtlMonitorModes (binary compatible layout)
            PCtlMonitorModes pModes = (PCtlMonitorModes)Buffer;
            Status = pContext->pContext->UpdateMonitorModes(pModes);
        }
        break;
    case IOCTL_QUICKDESK_VD_QUERY:
        {
            // Return active monitor status via output buffer
            PVOID OutBuffer;
            size_t OutBufSize;
            Status = WdfRequestRetrieveOutputBuffer(Request, sizeof(QuickDeskQueryResult), &OutBuffer, &OutBufSize);
            if (NT_SUCCESS(Status))
            {
                PQuickDeskQueryResult pResult = (PQuickDeskQueryResult)OutBuffer;
                memset(pResult, 0, sizeof(QuickDeskQueryResult));
                pResult->ActiveCount = 0;
                for (UINT i = 0; i < QUICKDESK_VD_MAX_MONITORS && i < pContext->pContext->GetMaxMonitorCount(); i++)
                {
                    pResult->Monitors[i].ConnectorIndex = i;
                    // Check if monitor is active by trying to access the monitor array
                    // (simplified - in production, expose a proper query method)
                    pResult->Monitors[i].Active = FALSE; // Will be set by enhanced CheckIndex
                }
                WdfRequestSetInformation(Request, sizeof(QuickDeskQueryResult));
            }
        }
        break;
    default:
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! unknown io ctrl code %ud",
            IoControlCode);
        Status = STATUS_NOT_IMPLEMENTED;
        break;
    }

    TraceEvents(TRACE_LEVEL_INFORMATION,
        TRACE_DEVICE,
        "%!FUNC! io control code %ul, %!STATUS!\n",
        IoControlCode, Status);

    WdfRequestComplete(Request, Status);
}

// TODO: This function may not be called, why?
_Use_decl_annotations_
NTSTATUS IddRustDeskAdapterInitFinished(IDDCX_ADAPTER AdapterObject, const IDARG_IN_ADAPTER_INIT_FINISHED* pInArgs)
{
    TraceEvents(TRACE_LEVEL_VERBOSE, TRACE_DEVICE, "%!FUNC! called");

    // This is called when the OS has finished setting up the adapter for use by the IddCx driver. It's now possible
    // to report attached monitors.

    auto* pDeviceContextWrapper = WdfObjectGet_IndirectDeviceContextWrapper(AdapterObject);
    auto Status = pInArgs->AdapterInitStatus;
    if (NT_SUCCESS(Status))
    {
        TraceEvents(TRACE_LEVEL_INFORMATION,
            TRACE_DEVICE,
            "%!FUNC! adapter init finished success");

        //for (DWORD i = 0; i < IDD_SAMPLE_MONITOR_COUNT; i++)
        //{
        //    pDeviceContextWrapper->pContext->FinishInit(i);
        //}
    }
    else
    {
        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! adapter init finished failed %!STATUS!",
            Status);
    }

    pDeviceContextWrapper->pContext->SetAdapterInitStatus(Status);
    return STATUS_SUCCESS;
}

_Use_decl_annotations_
NTSTATUS IddRustDeskAdapterCommitModes(IDDCX_ADAPTER AdapterObject, const IDARG_IN_COMMITMODES* pInArgs)
{
    TraceEvents(TRACE_LEVEL_VERBOSE, TRACE_DEVICE, "%!FUNC! called");

    UNREFERENCED_PARAMETER(AdapterObject);
    UNREFERENCED_PARAMETER(pInArgs);

    // For the sample, do nothing when modes are picked - the swap-chain is taken care of by IddCx

    // ==============================
    // TODO: In a real driver, this function would be used to reconfigure the device to commit the new modes. Loop
    // through pInArgs->pPaths and look for IDDCX_PATH_FLAGS_ACTIVE. Any path not active is inactive (e.g. the monitor
    // should be turned off).
    // ==============================

    return STATUS_SUCCESS;
}

_Use_decl_annotations_
NTSTATUS IddRustDeskParseMonitorDescription(const IDARG_IN_PARSEMONITORDESCRIPTION* pInArgs, IDARG_OUT_PARSEMONITORDESCRIPTION* pOutArgs)
{
    TraceEvents(TRACE_LEVEL_VERBOSE, TRACE_DEVICE, "%!FUNC! called");

    // ==============================
    // TODO: In a real driver, this function would be called to generate monitor modes for an EDID by parsing it. In
    // this sample driver, we hard-code the EDID, so this function can generate known modes.
    // ==============================

    pOutArgs->MonitorModeBufferOutputCount = IndirectSampleMonitor::szModeList;

    if (pInArgs->MonitorModeBufferInputCount < IndirectSampleMonitor::szModeList)
    {
        // Return success if there was no buffer, since the caller was only asking for a count of modes
        return (pInArgs->MonitorModeBufferInputCount > 0) ? STATUS_BUFFER_TOO_SMALL : STATUS_SUCCESS;
    }
    else
    {
        // In the sample driver, we have reported some static information about connected monitors
        // Check which of the reported monitors this call is for by comparing it to the pointer of
        // our known EDID blocks.

        if (pInArgs->MonitorDescription.DataSize != IndirectSampleMonitor::szEdidBlock)
            return STATUS_INVALID_PARAMETER;

        DWORD SampleMonitorIdx = 0;
        for(; SampleMonitorIdx < ARRAYSIZE(s_SampleMonitors); SampleMonitorIdx++)
        {
            if (memcmp(pInArgs->MonitorDescription.pData, s_SampleMonitors[SampleMonitorIdx].pEdidBlock, IndirectSampleMonitor::szEdidBlock) == 0)
            {
                // Copy the known modes to the output buffer
                for (DWORD ModeIndex = 0; ModeIndex < IndirectSampleMonitor::szModeList; ModeIndex++)
                {
                    pInArgs->pMonitorModes[ModeIndex] = CreateIddCxMonitorMode(
                        s_SampleMonitors[SampleMonitorIdx].pModeList[ModeIndex].Width,
                        s_SampleMonitors[SampleMonitorIdx].pModeList[ModeIndex].Height,
                        s_SampleMonitors[SampleMonitorIdx].pModeList[ModeIndex].VSync,
                        IDDCX_MONITOR_MODE_ORIGIN_MONITORDESCRIPTOR
                    );
                }

                // Set the preferred mode as represented in the EDID
                pOutArgs->PreferredMonitorModeIdx = s_SampleMonitors[SampleMonitorIdx].ulPreferredModeIdx;
        
                return STATUS_SUCCESS;
            }
        }

        TraceEvents(TRACE_LEVEL_ERROR,
            TRACE_DEVICE,
            "%!FUNC! invalid parameter");
        // This EDID block does not belong to the monitors we reported earlier
        return STATUS_INVALID_PARAMETER;
    }
}

_Use_decl_annotations_
NTSTATUS IddRustDeskMonitorGetDefaultModes(IDDCX_MONITOR MonitorObject, const IDARG_IN_GETDEFAULTDESCRIPTIONMODES* pInArgs, IDARG_OUT_GETDEFAULTDESCRIPTIONMODES* pOutArgs)
{
    TraceEvents(TRACE_LEVEL_VERBOSE, TRACE_DEVICE, "%!FUNC! called");

    UNREFERENCED_PARAMETER(MonitorObject);

    // ==============================
    // TODO: In a real driver, this function would be called to generate monitor modes for a monitor with no EDID.
    // Drivers should report modes that are guaranteed to be supported by the transport protocol and by nearly all
    // monitors (such 640x480, 800x600, or 1024x768). If the driver has access to monitor modes from a descriptor other
    // than an EDID, those modes would also be reported here.
    // ==============================

    if (pInArgs->DefaultMonitorModeBufferInputCount == 0)
    {
        pOutArgs->DefaultMonitorModeBufferOutputCount = ARRAYSIZE(s_SampleDefaultModes); 
    }
    else
    {
        for (DWORD ModeIndex = 0; ModeIndex < ARRAYSIZE(s_SampleDefaultModes); ModeIndex++)
        {
            pInArgs->pDefaultMonitorModes[ModeIndex] = CreateIddCxMonitorMode(
                s_SampleDefaultModes[ModeIndex].Width,
                s_SampleDefaultModes[ModeIndex].Height,
                s_SampleDefaultModes[ModeIndex].VSync,
                IDDCX_MONITOR_MODE_ORIGIN_DRIVER
            );
        }

        pOutArgs->DefaultMonitorModeBufferOutputCount = ARRAYSIZE(s_SampleDefaultModes); 
        pOutArgs->PreferredMonitorModeIdx = 0;
    }

    return STATUS_SUCCESS;
}

// more query modes
// https://github.com/roshkins/IddSampleDriver/blob/df7238c1f242e1093cdcab0ea749f34094570283/IddSampleDriver/Driver.cpp#L699
_Use_decl_annotations_
NTSTATUS IddRustDeskMonitorQueryModes(IDDCX_MONITOR MonitorObject, const IDARG_IN_QUERYTARGETMODES* pInArgs, IDARG_OUT_QUERYTARGETMODES* pOutArgs)
{
    TraceEvents(TRACE_LEVEL_VERBOSE, TRACE_DEVICE, "%!FUNC! called");

    UNREFERENCED_PARAMETER(MonitorObject);

    vector<IDDCX_TARGET_MODE> TargetModes;

    // Create a set of modes supported for frame processing and scan-out. These are typically not based on the
    // monitor's descriptor and instead are based on the static processing capability of the device. The OS will
    // report the available set of modes for a given output as the intersection of monitor modes with target modes.

    TargetModes.push_back(CreateIddCxTargetMode(3840, 2160, 60));
    TargetModes.push_back(CreateIddCxTargetMode(2560, 1440, 144));
    TargetModes.push_back(CreateIddCxTargetMode(2560, 1440, 90));
    TargetModes.push_back(CreateIddCxTargetMode(2560, 1440, 60));
    TargetModes.push_back(CreateIddCxTargetMode(1920, 1080, 144));
    TargetModes.push_back(CreateIddCxTargetMode(1920, 1080, 90));
    TargetModes.push_back(CreateIddCxTargetMode(1920, 1080, 60));
    TargetModes.push_back(CreateIddCxTargetMode(1600,  900, 60));
    TargetModes.push_back(CreateIddCxTargetMode(1024,  768, 75));
    TargetModes.push_back(CreateIddCxTargetMode(1024,  768, 60));

    pOutArgs->TargetModeBufferOutputCount = (UINT) TargetModes.size();

    if (pInArgs->TargetModeBufferInputCount >= TargetModes.size())
    {
        copy(TargetModes.begin(), TargetModes.end(), pInArgs->pTargetModes);
    }

    return STATUS_SUCCESS;
}

_Use_decl_annotations_
NTSTATUS IddRustDeskMonitorAssignSwapChain(IDDCX_MONITOR MonitorObject, const IDARG_IN_SETSWAPCHAIN* pInArgs)
{
    TraceEvents(TRACE_LEVEL_RESERVED7, TRACE_DEVICE, "%!FUNC! called");

    auto* pMonitorContextWrapper = WdfObjectGet_IndirectMonitorContextWrapper(MonitorObject);
    pMonitorContextWrapper->pContext->AssignSwapChain(pInArgs->hSwapChain, pInArgs->RenderAdapterLuid, pInArgs->hNextSurfaceAvailable);
    return STATUS_SUCCESS;
}

_Use_decl_annotations_
NTSTATUS IddRustDeskMonitorUnassignSwapChain(IDDCX_MONITOR MonitorObject)
{
    TraceEvents(TRACE_LEVEL_VERBOSE, TRACE_DEVICE, "%!FUNC! called");

    auto* pMonitorContextWrapper = WdfObjectGet_IndirectMonitorContextWrapper(MonitorObject);
    pMonitorContextWrapper->pContext->UnassignSwapChain();
    return STATUS_SUCCESS;
}

#pragma endregion
