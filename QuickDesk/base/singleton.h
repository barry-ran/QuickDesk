#pragma once

namespace base {

template <class T>
class Singleton {
public:
    static T& instance()
    {
        if (!s_ins) {
            s_ins = new T();
        }
        return *s_ins;
    }
    Singleton(const Singleton&) = delete;
    Singleton& operator=(const Singleton&) = delete;
    virtual ~Singleton() { }

protected:
    Singleton() { }

private:
    static T* s_ins;
};

template <class T>
T* Singleton<T>::s_ins = nullptr;
}
